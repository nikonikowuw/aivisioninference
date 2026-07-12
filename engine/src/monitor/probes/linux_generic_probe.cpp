#include "probes/device_probe.h"

#include <unistd.h>
#include <dirent.h>
#include <sys/utsname.h>
#include <sys/statvfs.h>
#ifndef __APPLE__
#include <sys/sysinfo.h>
#endif
#include <fstream>
#include <sstream>
#include <iostream>
#include <algorithm>
#include <cctype>
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

                // 3. Storage Usage (single path, kept for backward compat with EdgeNode metrics)
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
                        uint64_t free_blocks = vfs.f_bavail;
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

            void CollectExtendedSnapshot(
                const DeviceMonitorConfig& config,
                DeviceSnapshot& snapshot,
                uint64_t now_ms) override
            {
#ifndef __APPLE__
                // 1. Load Average from /proc/loadavg
                {
                    std::ifstream loadavg(config.proc_dir + "/loadavg");
                    if (loadavg.is_open())
                    {
                        std::string line;
                        if (std::getline(loadavg, line))
                        {
                            std::stringstream ss(line);
                            double la1 = 0, la5 = 0, la15 = 0;
                            ss >> la1 >> la5 >> la15;

                            auto setLoad = [&](MetricValue& mv, double val, const char* field) {
                                mv.available = true;
                                mv.stale = false;
                                mv.value = val;
                                mv.unit = "load";
                                mv.source = config.proc_dir + "/loadavg (" + field + ")";
                                mv.collected_at_ms = now_ms;
                            };
                            setLoad(snapshot.metrics.load_average_1m, la1, "1m");
                            setLoad(snapshot.metrics.load_average_5m, la5, "5m");
                            setLoad(snapshot.metrics.load_average_15m, la15, "15m");
                        }
                    }
                }

                // 2. Current process and thread counts from numeric /proc entries.
                {
                    uint64_t process_count = 0;
                    uint64_t thread_count = 0;
                    DIR* proc_dir = opendir(config.proc_dir.c_str());
                    if (proc_dir != nullptr)
                    {
                        while (const dirent* entry = readdir(proc_dir))
                        {
                            const std::string name(entry->d_name);
                            if (name.empty() || !std::all_of(name.begin(), name.end(), [](unsigned char ch) {
                                    return std::isdigit(ch) != 0;
                                }))
                            {
                                continue;
                            }

                            ++process_count;
                            const std::string task_path = config.proc_dir + "/" + name + "/task";
                            DIR* task_dir = opendir(task_path.c_str());
                            if (task_dir == nullptr)
                            {
                                continue;
                            }
                            while (const dirent* task_entry = readdir(task_dir))
                            {
                                const std::string task_name(task_entry->d_name);
                                if (!task_name.empty() && std::all_of(task_name.begin(), task_name.end(), [](unsigned char ch) {
                                        return std::isdigit(ch) != 0;
                                    }))
                                {
                                    ++thread_count;
                                }
                            }
                            closedir(task_dir);
                        }
                        closedir(proc_dir);

                        auto setCount = [now_ms, &config](MetricValue& metric, uint64_t value, const char* field) {
                            metric.available = true;
                            metric.stale = false;
                            metric.value = static_cast<double>(value);
                            metric.unit = "count";
                            metric.source = config.proc_dir + " numeric entries (" + field + ")";
                            metric.error.clear();
                            metric.collected_at_ms = now_ms;
                        };
                        setCount(snapshot.metrics.process_count, process_count, "processes");
                        setCount(snapshot.metrics.thread_count, thread_count, "threads");
                    }
                }

                // 3. Network I/O from /proc/net/dev
                {
                    std::ifstream net_file(config.proc_dir + "/net/dev");
                    if (net_file.is_open())
                    {
                        std::string line;
                        // Skip two header lines
                        std::getline(net_file, line);
                        std::getline(net_file, line);

                        uint64_t total_rx = 0;
                        uint64_t total_tx = 0;

                        while (std::getline(net_file, line))
                        {
                            // Format: "interfacename: RX_bytes RX_packets ... TX_bytes TX_packets ..."
                            size_t colon = line.find(':');
                            if (colon == std::string::npos) continue;

                            std::string iface = line.substr(0, colon);
                            iface.erase(0, iface.find_first_not_of(" \t"));
                            iface.erase(iface.find_last_not_of(" \t") + 1);
                            // Exclude only loopback and IFB redirect devices.
                            if (iface == "lo" || iface.rfind("ifb", 0) == 0) continue;

                            std::string stats = line.substr(colon + 1);
                            std::stringstream ss(stats);
                            uint64_t rx_bytes = 0, rx_packets = 0;
                            uint64_t tx_bytes = 0, tx_packets = 0;
                            // Skip: RX bytes, packets, errs, drop, fifo, frame, compressed, multicast
                            // Then TX: bytes, packets, errs, drop, fifo, colls, carrier, compressed
                            uint64_t temp = 0;
                            ss >> rx_bytes >> rx_packets;
                            for (int i = 0; i < 6; ++i) ss >> temp;
                            ss >> tx_bytes >> tx_packets;

                            total_rx += rx_bytes;
                            total_tx += tx_bytes;
                        }

                        snapshot.network.rx_bytes = total_rx;
                        snapshot.network.tx_bytes = total_tx;

                        // Compute network speed via delta tracking
                        uint64_t now_us = std::chrono::duration_cast<std::chrono::microseconds>(
                            std::chrono::steady_clock::now().time_since_epoch()).count();

                        if (last_net_sample_us_ > 0) {
                            double elapsed_sec = (now_us - last_net_sample_us_) / 1e6;
                            if (elapsed_sec > 0 && total_rx >= last_net_rx_bytes_ && total_tx >= last_net_tx_bytes_) {
                                snapshot.network.rx_speed = (total_rx - last_net_rx_bytes_) / elapsed_sec;
                                snapshot.network.tx_speed = (total_tx - last_net_tx_bytes_) / elapsed_sec;
                            } else {
                                // Counter resets or disappearing interfaces start a new baseline.
                                snapshot.network.rx_speed = 0.0;
                                snapshot.network.tx_speed = 0.0;
                            }
                        }

                        last_net_rx_bytes_ = total_rx;
                        last_net_tx_bytes_ = total_tx;
                        last_net_sample_us_ = now_us;
                    }
                }

                // 4. Per-mountpoint disk usage from /proc/mounts + statvfs
                {
                    snapshot.disks.clear();

                    auto addDiskMetric = [&](const std::string& mnt_path) {
                        struct statvfs dvfs;
                        if (statvfs(mnt_path.c_str(), &dvfs) != 0) return;
                        if (dvfs.f_blocks == 0) return;

                        DiskInfo disk;
                        disk.path = mnt_path;
                        disk.total_bytes = static_cast<uint64_t>(dvfs.f_blocks) * dvfs.f_frsize;
                        disk.used_bytes = (static_cast<uint64_t>(dvfs.f_blocks) - static_cast<uint64_t>(dvfs.f_bavail)) * dvfs.f_frsize;
                        snapshot.disks.push_back(disk);
                    };

                    // Collect root first (most important)
                    addDiskMetric("/");

                    // Scan /proc/mounts for other physical mount points
                    std::ifstream mounts(config.proc_dir + "/mounts");
                    if (mounts.is_open())
                    {
                        std::string line;
                        while (std::getline(mounts, line))
                        {
                            std::stringstream ss(line);
                            std::string dev, mnt, fstype, opts;
                            ss >> dev >> mnt >> fstype >> opts;

                            // Skip non-physical filesystems
                            if (dev.empty() || dev.rfind("/dev/", 0) != 0) continue;
                            // Skip pseudo filesystems
                            if (fstype == "proc" || fstype == "sysfs" || fstype == "tmpfs" ||
                                fstype == "devtmpfs" || fstype == "devpts" || fstype == "cgroup" ||
                                fstype == "cgroup2" || fstype == "pstore" || fstype == "securityfs" ||
                                fstype == "selinuxfs" || fstype == "hugetlbfs" || fstype == "mqueue" ||
                                fstype == "debugfs" || fstype == "tracefs" || fstype == "configfs" ||
                                fstype == "efivarfs" || fstype == "bpf" || fstype == "fuse" ||
                                fstype == "autofs" || fstype == "overlay") continue;

                            // Skip root (already added) and duplicates
                            if (mnt == "/") continue;

                            bool already = false;
                            for (const auto& d : snapshot.disks) {
                                if (d.path == mnt) { already = true; break; }
                            }
                            if (already) continue;

                            addDiskMetric(mnt);
                        }
                    }
                }
#endif
            }

        private:
            bool has_last_cpu_ = false;
            uint64_t last_cpu_total_ = 0;
            uint64_t last_cpu_idle_ = 0;

            // Network delta tracking
            uint64_t last_net_rx_bytes_ = 0;
            uint64_t last_net_tx_bytes_ = 0;
            uint64_t last_net_sample_us_ = 0;
        };

        std::shared_ptr<IDeviceProbe> CreateLinuxGenericProbe()
        {
            return std::make_shared<LinuxGenericProbe>();
        }
    }
}
