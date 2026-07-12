#include "monitor/device_monitor.h"
#include "probes/device_probe.h"

#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/select.h>
#include <signal.h>
#include <fcntl.h>
#include <cstring>
#include <vector>
#include <string>
#include <iostream>
#include <chrono>
#include <algorithm>
#include <unordered_map>

namespace aivision
{
    namespace monitor
    {
        namespace
        {
            uint64_t GetCurrentTimeMs()
            {
                return std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch()).count();
            }

            class CommandBackoff
            {
            public:
                CommandBackoff() = default;

                bool CanRun(uint64_t now_ms) const
                {
                    return now_ms >= next_allowed_run_ms_;
                }

                void RecordSuccess()
                {
                    failure_count_ = 0;
                    next_allowed_run_ms_ = 0;
                }

                void RecordFailure(uint64_t now_ms)
                {
                    failure_count_++;
                    uint32_t delay_seconds = 30;
                    if (failure_count_ == 1) delay_seconds = 30;
                    else if (failure_count_ == 2) delay_seconds = 60;
                    else if (failure_count_ == 3) delay_seconds = 300;
                    else delay_seconds = 600;

                    next_allowed_run_ms_ = now_ms + delay_seconds * 1000;
                }

            private:
                uint32_t failure_count_ = 0;
                uint64_t next_allowed_run_ms_ = 0;
            };
        } // namespace

        // ============================================================
        // DeviceMonitor Implementation
        // ============================================================

        DeviceMonitor::DeviceMonitor(const DeviceMonitorConfig& config)
            : config_(config)
        {
        }

        DeviceMonitor::~DeviceMonitor()
        {
            Stop();
        }

        std::shared_ptr<const DeviceSnapshot> DeviceMonitor::GetSnapshotPtr()
        {
            std::lock_guard<std::mutex> lock(mutex_);
            return cached_snapshot_ptr_;
        }

        bool DeviceMonitor::Initialize()
        {
            try
            {
                // Instantiate all probes
                probes_.push_back(CreateLinuxGenericProbe());
                probes_.push_back(CreateRockchipProbe());
                probes_.push_back(CreateNvidiaProbe());
                probes_.push_back(CreateAscendProbe());
                probes_.push_back(CreateMacOSProbe());

                std::string override_platform = config_.platform_override;
                std::transform(override_platform.begin(), override_platform.end(), override_platform.begin(), ::tolower);

                for (auto& probe : probes_)
                {
                    if (!probe) continue;

                    bool active = false;
                    std::vector<std::string> evidence;

                    std::string name = probe->Name();
                    if (!override_platform.empty() && name.find(override_platform) != std::string::npos)
                    {
                        active = true;
                        evidence.push_back("Force-activated via platform override: " + config_.platform_override);
                    }
                    else
                    {
                        auto detect = probe->Detect(config_);
                        if (detect.supported && detect.confidence >= 70)
                        {
                            active = true;
                            evidence = detect.evidence;
                        }
                    }

                    if (active)
                    {
                        active_probes_.push_back(probe);
                    }
                }

                // Publish initial default unavailable snapshot
                DeviceSnapshot initial;
                initial.timestamp_ms = GetCurrentTimeMs();
                // Ensure all fields have diagnostics indicating init status
                for (auto& probe : active_probes_)
                {
                    ProbeDiagnostic diag;
                    diag.name = probe->Name();
                    diag.success = true;
                    diag.evidence = {"Initialized, pending first snapshot collection"};
                    initial.diagnostics.push_back(diag);
                }
                PublishSnapshot(initial);

                return !active_probes_.empty();
            }
            catch (const std::exception& e)
            {
                std::cerr << "DeviceMonitor initialize error: " << e.what() << std::endl;
                DeviceSnapshot initial;
                initial.timestamp_ms = GetCurrentTimeMs();
                PublishSnapshot(initial);
                return false;
            }
            catch (...)
            {
                std::cerr << "DeviceMonitor initialize unknown error" << std::endl;
                DeviceSnapshot initial;
                initial.timestamp_ms = GetCurrentTimeMs();
                PublishSnapshot(initial);
                return false;
            }
        }

        void DeviceMonitor::Start()
        {
            if (running_.load())
                return;

            running_.store(true);
            thread_ = std::make_unique<std::thread>(&DeviceMonitor::MonitorLoop, this);
        }

        void DeviceMonitor::Stop()
        {
            if (!running_.load())
                return;

            running_.store(false);
            if (thread_ && thread_->joinable())
            {
                thread_->join();
            }
            thread_.reset();
        }

        void DeviceMonitor::PublishSnapshot(const DeviceSnapshot& new_snapshot)
        {
            auto ptr = std::make_shared<DeviceSnapshot>(new_snapshot);
            std::lock_guard<std::mutex> lock(mutex_);
            cached_snapshot_ptr_ = std::move(ptr);
        }

        DeviceSnapshot DeviceMonitor::GetSnapshot()
        {
            std::lock_guard<std::mutex> lock(mutex_);
            if (!cached_snapshot_ptr_) return DeviceSnapshot{};
            return *cached_snapshot_ptr_;
        }

        void DeviceMonitor::MonitorLoop()
        {
            auto last_static_run = std::chrono::steady_clock::now() - std::chrono::hours(1);
            auto last_light_run = std::chrono::steady_clock::now() - std::chrono::hours(1);
            auto last_expensive_run = std::chrono::steady_clock::now() - std::chrono::hours(1);

            std::unordered_map<std::string, CommandBackoff> probe_backoffs;

            while (running_.load())
            {
                try
                {
                    auto now = std::chrono::steady_clock::now();
                    uint64_t now_ms = GetCurrentTimeMs();

                    bool changed = false;
                    DeviceSnapshot next_snapshot = GetSnapshot();
                    next_snapshot.timestamp_ms = now_ms;

                    // 1. Static Info
                    bool run_static = (std::chrono::duration_cast<std::chrono::milliseconds>(now - last_static_run).count() >= 600000);
                    if (run_static)
                    {
                        for (auto& probe : active_probes_)
                        {
                            try
                            {
                                probe->CollectStaticInfo(config_, next_snapshot.info);
                            }
                            catch (const std::exception& e)
                            {
                                std::cerr << "Probe " << probe->Name() << " CollectStaticInfo error: " << e.what() << std::endl;
                            }
                        }
                        last_static_run = now;
                        changed = true;
                    }

                    // 2. Light metrics
                    bool run_light = (std::chrono::duration_cast<std::chrono::milliseconds>(now - last_light_run).count() >= config_.light_probe_interval_ms);
                    if (run_light)
                    {
                        for (auto& probe : active_probes_)
                        {
                            try
                            {
                                probe->CollectDynamicMetrics(config_, next_snapshot.metrics, next_snapshot.accelerators);
                            }
                            catch (const std::exception& e)
                            {
                                std::cerr << "Probe " << probe->Name() << " CollectDynamicMetrics error: " << e.what() << std::endl;
                            }
                        }
                        last_light_run = now;
                        changed = true;
                    }

                    // 3. Expensive metrics
                    bool run_expensive = (std::chrono::duration_cast<std::chrono::milliseconds>(now - last_expensive_run).count() >= config_.expensive_probe_interval_ms);
                    if (run_expensive)
                    {
                        for (auto& probe : active_probes_)
                        {
                            std::string name = probe->Name();
                            auto& backoff = probe_backoffs[name];
                            if (backoff.CanRun(now_ms))
                            {
                                try
                                {
                                    probe->CollectExpensiveMetrics(config_, next_snapshot.metrics, next_snapshot.accelerators);
                                    backoff.RecordSuccess();
                                }
                                catch (const std::exception& e)
                                {
                                    std::cerr << "Probe " << name << " CollectExpensiveMetrics error: " << e.what() << std::endl;
                                    backoff.RecordFailure(now_ms);
                                }
                            }
                        }
                        last_expensive_run = now;
                        changed = true;
                    }

                    // 4. Extended snapshot fields (disk mountpoints, network I/O, load, process/thread)
                    if (run_light || run_expensive)
                    {
                        for (auto& probe : active_probes_)
                        {
                            try
                            {
                                probe->CollectExtendedSnapshot(config_, next_snapshot, now_ms);
                            }
                            catch (const std::exception& e)
                            {
                                std::cerr << "Probe " << probe->Name() << " CollectExtendedSnapshot error: " << e.what() << std::endl;
                            }
                        }
                        changed = true;
                    }

                    if (changed)
                    {
                        next_snapshot.diagnostics.clear();
                        for (auto& probe : active_probes_)
                        {
                            ProbeDiagnostic diag;
                            diag.name = probe->Name();
                            diag.success = true;
                            auto detect = probe->Detect(config_);
                            diag.evidence = detect.evidence;
                            next_snapshot.diagnostics.push_back(diag);
                        }

                        PublishSnapshot(next_snapshot);
                    }
                }
                catch (const std::exception& e)
                {
                    std::cerr << "DeviceMonitor loop error: " << e.what() << std::endl;
                }
                catch (...)
                {
                    std::cerr << "DeviceMonitor loop unknown error" << std::endl;
                }

                // Sleep in small increments to allow fast shutdown response
                for (int i = 0; i < 50 && running_.load(); ++i)
                {
                    std::this_thread::sleep_for(std::chrono::milliseconds(100));
                }
            }
        }
    }
}
