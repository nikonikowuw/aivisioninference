#include "probes/device_probe.h"

#include <unistd.h>
#include <sys/utsname.h>
#include <sys/statvfs.h>
#ifndef __APPLE__
#include <sys/sysinfo.h>
#endif
#include <fstream>
#include <sstream>
#include <iostream>
#include <algorithm>
#include <chrono>

namespace aivision
{
    namespace monitor
    {
        class LinuxGenericProbe : public IDeviceProbe
        {
        public:
            LinuxGenericProbe() = default;
            ~LinuxGenericProbe() override = default;

            std::string Name() const override { return "linux_generic"; }

            ProbeDetection Detect(const DeviceMonitorConfig& config) override
            {
                ProbeDetection det;
#ifdef __APPLE__
                det.supported = false;
                det.confidence = 0;
                det.evidence.push_back("Platform is macOS (Darwin)");
#else
                struct utsname uts;
                if (uname(&uts) == 0 && std::string(uts.sysname) == "Linux")
                {
                    det.supported = true;
                    det.confidence = 100;
                    det.evidence.push_back("OS is Linux");
                    det.evidence.push_back("Kernel: " + std::string(uts.release));
                    det.evidence.push_back("Arch: " + std::string(uts.machine));
                }
                else
                {
                    det.supported = false;
                    det.confidence = 0;
                    det.evidence.push_back("Unknown non-Linux platform");
                }
#endif
                return det;
            }

            void CollectStaticInfo(const DeviceMonitorConfig& config, DeviceStaticInfo& info) override
            {
#ifndef __APPLE__
                // Hostname
                char hostname_buf[256];
                if (gethostname(hostname_buf, sizeof(hostname_buf)) == 0)
                {
                    info.hostname = hostname_buf;
                }

                // OS / Kernel / Arch
                struct utsname uts;
                if (uname(&uts) == 0)
                {
                    info.os = "linux";
                    info.kernel = uts.release;
                    info.arch = uts.machine;
                }

                // CPU cores & CPU Model
                int cores = 0;
                std::string cpu_model_name;
                std::string cpu_hardware;
                std::ifstream cpuinfo(config.proc_dir + "/cpuinfo");
                if (cpuinfo.is_open())
                {
                    std::string line;
                    while (std::getline(cpuinfo, line))
                    {
                        if (line.rfind("processor", 0) == 0)
                        {
                            cores++;
                        }
                        else if (cpu_model_name.empty() && line.rfind("model name", 0) == 0)
                        {
                            size_t colon = line.find(":");
                            if (colon != std::string::npos)
                            {
                                cpu_model_name = line.substr(colon + 1);
                                // Trim spaces
                                cpu_model_name.erase(0, cpu_model_name.find_first_not_of(" \t"));
                                cpu_model_name.erase(cpu_model_name.find_last_not_of(" \t") + 1);
                            }
                        }
                        else if (cpu_hardware.empty() && line.rfind("Hardware", 0) == 0)
                        {
                            size_t colon = line.find(":");
                            if (colon != std::string::npos)
                            {
                                cpu_hardware = line.substr(colon + 1);
                                cpu_hardware.erase(0, cpu_hardware.find_first_not_of(" \t"));
                                cpu_hardware.erase(cpu_hardware.find_last_not_of(" \t") + 1);
                            }
                        }
                    }
                }

                info.cpu_cores = cores > 0 ? cores : sysconf(_SC_NPROCESSORS_CONF);

                if (!cpu_model_name.empty())
                {
                    info.cpu_model = cpu_model_name;
                }
                else if (!cpu_hardware.empty())
                {
                    info.cpu_model = cpu_hardware;
                }
                else
                {
                    info.cpu_model = info.arch;
                }

                // Total Memory
                std::ifstream meminfo(config.proc_dir + "/meminfo");
                if (meminfo.is_open())
                {
                    std::string line;
                    while (std::getline(meminfo, line))
                    {
                        if (line.rfind("MemTotal:", 0) == 0)
                        {
                            std::stringstream ss(line);
                            std::string label;
                            uint64_t val = 0;
                            std::string unit;
                            ss >> label >> val >> unit;
                            if (unit == "kB")
                            {
                                info.total_memory = val * 1024;
                            }
                            else
                            {
                                info.total_memory = val;
                            }
                            break;
                        }
                    }
                }

                // Total Storage
                std::string path = config.storage_path;
                if (path.empty()) path = "/";
                struct statvfs vfs;
                if (statvfs(path.c_str(), &vfs) == 0)
                {
                    info.total_storage = static_cast<uint64_t>(vfs.f_blocks) * vfs.f_frsize;
                }
#endif
            }

            void CollectDynamicMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
#ifndef __APPLE__
                uint64_t now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch()).count();

                // 1. CPU Usage
                std::ifstream stat_file(config.proc_dir + "/stat");
                if (stat_file.is_open())
                {
                    std::string line;
                    if (std::getline(stat_file, line) && line.rfind("cpu ", 0) == 0)
                    {
                        std::stringstream ss(line);
                        std::string cpu_label;
                        uint64_t user = 0, nice = 0, system = 0, idle = 0, iowait = 0;
                        uint64_t irq = 0, softirq = 0, steal = 0, guest = 0, guest_nice = 0;
                        ss >> cpu_label >> user >> nice >> system >> idle >> iowait >> irq >> softirq >> steal >> guest >> guest_nice;

                        uint64_t total_idle = idle + iowait;
                        uint64_t total_non_idle = user + nice + system + irq + softirq + steal;
                        uint64_t total = total_idle + total_non_idle;

                        if (has_last_cpu_)
                        {
                            uint64_t delta_total = total - last_cpu_total_;
                            uint64_t delta_idle = total_idle - last_cpu_idle_;

                            if (delta_total > 0 && delta_total >= delta_idle)
                            {
                                metrics.cpu_usage.available = true;
                                metrics.cpu_usage.stale = false;
                                metrics.cpu_usage.value = (1.0 - static_cast<double>(delta_idle) / delta_total) * 100.0;
                                metrics.cpu_usage.unit = "percent";
                                metrics.cpu_usage.source = config.proc_dir + "/stat";
                                metrics.cpu_usage.error = "";
                                metrics.cpu_usage.collected_at_ms = now_ms;
                            }
                            else
                            {
                                metrics.cpu_usage.available = false;
                                metrics.cpu_usage.error = "Invalid stat delta values";
                            }
                        }
                        else
                        {
                            metrics.cpu_usage.available = false;
                            metrics.cpu_usage.error = "waiting for second sample";
                        }

                        last_cpu_total_ = total;
                        last_cpu_idle_ = total_idle;
                        has_last_cpu_ = true;
                    }
                }

                // 2. Memory Usage
                std::ifstream meminfo_file(config.proc_dir + "/meminfo");
                if (meminfo_file.is_open())
                {
                    std::string line;
                    uint64_t total_mem = 0;
                    uint64_t avail_mem = 0;
                    while (std::getline(meminfo_file, line))
                    {
                        if (line.rfind("MemTotal:", 0) == 0)
                        {
                            std::stringstream ss(line);
                            std::string label;
                            ss >> label >> total_mem;
                        }
                        else if (line.rfind("MemAvailable:", 0) == 0)
                        {
                            std::stringstream ss(line);
                            std::string label;
                            ss >> label >> avail_mem;
                        }
                    }

                    if (total_mem > 0)
                    {
                        metrics.memory_usage.available = true;
                        metrics.memory_usage.stale = false;
                        uint64_t used_mem = total_mem - avail_mem;
                        metrics.memory_usage.value = (static_cast<double>(used_mem) / total_mem) * 100.0;
                        metrics.memory_usage.unit = "percent";
                        metrics.memory_usage.source = config.proc_dir + "/meminfo";
                        metrics.memory_usage.error = "";
                        metrics.memory_usage.collected_at_ms = now_ms;
                    }
                    else
                    {
                        metrics.memory_usage.available = false;
                        metrics.memory_usage.error = "MemTotal parsing failed";
                    }
                }

                // 3. Storage Usage
                std::string path = config.storage_path;
                if (path.empty()) path = "/";
                struct statvfs vfs;
                if (statvfs(path.c_str(), &vfs) == 0)
                {
                    if (vfs.f_blocks > 0)
                    {
                        metrics.storage_usage.available = true;
                        metrics.storage_usage.stale = false;
                        uint64_t total_blocks = vfs.f_blocks;
                        uint64_t free_blocks = vfs.f_bavail; // blocks available to non-superuser
                        uint64_t used_blocks = total_blocks - free_blocks;
                        metrics.storage_usage.value = (static_cast<double>(used_blocks) / total_blocks) * 100.0;
                        metrics.storage_usage.unit = "percent";
                        metrics.storage_usage.source = "statvfs(" + path + ")";
                        metrics.storage_usage.error = "";
                        metrics.storage_usage.collected_at_ms = now_ms;
                    }
                    else
                    {
                        metrics.storage_usage.available = false;
                        metrics.storage_usage.error = "Zero blocks reported by filesystem";
                    }
                }
                else
                {
                    metrics.storage_usage.available = false;
                    metrics.storage_usage.error = "statvfs failed: " + std::string(strerror(errno));
                }
#endif
            }

            void CollectExpensiveMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
                // Linux generic doesn't need expensive collection
            }

        private:
            bool has_last_cpu_ = false;
            uint64_t last_cpu_total_ = 0;
            uint64_t last_cpu_idle_ = 0;
        };

        std::shared_ptr<IDeviceProbe> CreateLinuxGenericProbe()
        {
            return std::make_shared<LinuxGenericProbe>();
        }
    }
}
