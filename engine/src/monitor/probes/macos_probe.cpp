#include "probes/device_probe.h"

#include <unistd.h>
#include <sys/utsname.h>
#include <sys/statvfs.h>
#include <fstream>
#include <sstream>
#include <algorithm>
#include <chrono>
#include <iostream>
#include <set>

#ifdef __APPLE__
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <mach/mach.h>
#include <mach/vm_statistics.h>
#endif

namespace aivision
{
    namespace monitor
    {
        class MacOSProbe : public IDeviceProbe
        {
        public:
            MacOSProbe() = default;
            ~MacOSProbe() override = default;

            std::string Name() const override { return "macos"; }

            ProbeDetection Detect(const DeviceMonitorConfig& config) override
            {
                ProbeDetection det;
#ifdef __APPLE__
                struct utsname uts;
                if (uname(&uts) == 0 && std::string(uts.sysname) == "Darwin")
                {
                    det.supported = true;
                    det.confidence = 100;
                    det.evidence.push_back("OS is macOS (Darwin)");
                }
                else
                {
                    det.supported = false;
                    det.confidence = 0;
                    det.evidence.push_back("Non-Darwin apple platform");
                }
#else
                det.supported = false;
                det.confidence = 0;
                det.evidence.push_back("Platform is not macOS/Apple");
#endif
                return det;
            }

            void CollectStaticInfo(const DeviceMonitorConfig& config, DeviceStaticInfo& info) override
            {
#ifdef __APPLE__
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
                    info.os = "macos";
                    info.kernel = uts.release;
                    info.arch = uts.machine;
                }

                // CPU Model
                char cpu_model[256];
                size_t cpu_model_len = sizeof(cpu_model);
                if (sysctlbyname("machdep.cpu.brand_string", cpu_model, &cpu_model_len, NULL, 0) == 0)
                {
                    info.cpu_model = cpu_model;
                }
                else
                {
                    info.cpu_model = "Apple Silicon";
                }

                // CPU Cores
                int cores = 0;
                size_t cores_len = sizeof(cores);
                if (sysctlbyname("hw.ncpu", &cores, &cores_len, NULL, 0) == 0)
                {
                    info.cpu_cores = cores;
                }

                // Total Memory
                int64_t total_mem = 0;
                size_t total_mem_len = sizeof(total_mem);
                int mib[2] = {CTL_HW, HW_MEMSIZE};
                if (sysctl(mib, 2, &total_mem, &total_mem_len, NULL, 0) == 0)
                {
                    info.total_memory = total_mem;
                }

                // Total Storage
                std::string path = config.storage_path;
                if (path.empty()) path = "/";
                struct statvfs vfs;
                if (statvfs(path.c_str(), &vfs) == 0)
                {
                    info.total_storage = static_cast<uint64_t>(vfs.f_blocks) * vfs.f_frsize;
                }

                info.gpu_model = "Apple GPU";
                info.npu_model = "Apple Neural Engine";
                info.board = "Apple Mac";
#endif
            }

            void CollectDynamicMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
#ifdef __APPLE__
                uint64_t now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch()).count();

                mach_port_t host_port = mach_host_self();

                // 1. CPU Load
                host_cpu_load_info_data_t cpu_load;
                mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;
                if (host_statistics(host_port, HOST_CPU_LOAD_INFO, (host_info_t)&cpu_load, &count) == KERN_SUCCESS)
                {
                    uint64_t user = cpu_load.cpu_ticks[CPU_STATE_USER];
                    uint64_t system = cpu_load.cpu_ticks[CPU_STATE_SYSTEM];
                    uint64_t idle = cpu_load.cpu_ticks[CPU_STATE_IDLE];
                    uint64_t nice = cpu_load.cpu_ticks[CPU_STATE_NICE];

                    uint64_t total = user + system + idle + nice;
                    uint64_t used = user + system + nice;

                    if (has_last_cpu_)
                    {
                        uint64_t delta_total = total - last_cpu_total_;
                        uint64_t delta_used = used - last_cpu_used_;

                        if (delta_total > 0)
                        {
                            metrics.cpu_usage.available = true;
                            metrics.cpu_usage.stale = false;
                            metrics.cpu_usage.value = (static_cast<double>(delta_used) / delta_total) * 100.0;
                            metrics.cpu_usage.unit = "percent";
                            metrics.cpu_usage.source = "host_statistics(HOST_CPU_LOAD_INFO)";
                            metrics.cpu_usage.error = "";
                            metrics.cpu_usage.collected_at_ms = now_ms;
                        }
                    }
                    else
                    {
                        metrics.cpu_usage.available = false;
                        metrics.cpu_usage.error = "waiting for second sample";
                    }

                    last_cpu_total_ = total;
                    last_cpu_used_ = used;
                    has_last_cpu_ = true;
                }

                // 2. Memory Usage
                vm_size_t page_size;
                host_page_size(host_port, &page_size);

                vm_statistics64_data_t vm_stats = {};
                mach_msg_type_number_t vm_count = HOST_VM_INFO64_COUNT;
                if (host_statistics64(host_port, HOST_VM_INFO64, (host_info64_t)&vm_stats, &vm_count) == KERN_SUCCESS)
                {
                    // Matching Activity Monitor's "Memory Used":
                    //   active  — pages recently referenced, actively in use
                    //   wire    — pages pinned in RAM (cannot be paged out)
                    //   speculative — pages likely to be accessed soon
                    //   compressed — pages compressed by memory compressor (counted as used)
                    //   purgeable  — pages marked purgeable (reclaimable, deducted from used)
                    //
                    // Formula: used = (active + wire + speculative + compressed - purgeable)
                    //
                    // On older macOS kernels that don't fill compressor_page_count or
                    // purgeable_count, the zero-initialized struct makes these 0,
                    // gracefully degrading to the legacy formula.
                    // Use int64_t for used_pages to avoid unsigned underflow when
                    // purgeable_count exceeds the sum of the other counters.
                    int64_t used_pages = static_cast<int64_t>(vm_stats.active_count)
                                       + static_cast<int64_t>(vm_stats.wire_count)
                                       + static_cast<int64_t>(vm_stats.speculative_count)
                                       + static_cast<int64_t>(vm_stats.compressor_page_count)
                                       - static_cast<int64_t>(vm_stats.purgeable_count);
                    if (used_pages < 0) used_pages = 0;

                    int64_t total_mem = 0;
                    size_t total_mem_len = sizeof(total_mem);
                    int mib[2] = {CTL_HW, HW_MEMSIZE};
                    sysctl(mib, 2, &total_mem, &total_mem_len, NULL, 0);

                    if (total_mem > 0)
                    {
                        double pct = (static_cast<double>(used_pages) * page_size / total_mem) * 100.0;
                        if (pct > 100.0) pct = 100.0;

                        metrics.memory_usage.available = true;
                        metrics.memory_usage.stale = false;
                        metrics.memory_usage.value = pct;
                        metrics.memory_usage.unit = "percent";
                        metrics.memory_usage.source = "host_statistics64 (active+wire+spec+compressed-purgeable)";
                        metrics.memory_usage.error = "";
                        metrics.memory_usage.collected_at_ms = now_ms;
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
                        uint64_t free_blocks = vfs.f_bavail;
                        uint64_t used_blocks = total_blocks - free_blocks;

                        metrics.storage_usage.value = (static_cast<double>(used_blocks) / total_blocks) * 100.0;
                        metrics.storage_usage.unit = "percent";
                        metrics.storage_usage.source = "statvfs(" + path + ")";
                        metrics.storage_usage.error = "";
                        metrics.storage_usage.collected_at_ms = now_ms;
                    }
                }

                // 4. Apple GPU and NPU accelerator entries (usage unavailable)
                AcceleratorInfo gpu;
                gpu.type = "gpu";
                gpu.vendor = "apple";
                gpu.name = "Apple GPU";
                gpu.usage.available = false;
                gpu.usage.error = "Not supported on macOS";

                AcceleratorInfo npu;
                npu.type = "npu";
                npu.vendor = "apple";
                npu.name = "Apple Neural Engine";
                npu.usage.available = false;
                npu.usage.error = "Not supported on macOS";

                // Update accelerators list
                auto update_acc = [&](const AcceleratorInfo& acc) {
                    auto it = std::find_if(accelerators.begin(), accelerators.end(), [&](const AcceleratorInfo& a) {
                        return a.vendor == acc.vendor && a.type == acc.type;
                    });
                    if (it != accelerators.end())
                    {
                        *it = acc;
                    }
                    else
                    {
                        accelerators.push_back(acc);
                    }
                };

                update_acc(gpu);
                update_acc(npu);

                metrics.gpu_usage.available = false;
                metrics.gpu_usage.error = "Not supported on macOS";
                metrics.npu_usage.available = false;
                metrics.npu_usage.error = "Not supported on macOS";
#endif
            }

            void CollectExpensiveMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
                // macOS doesn't have expensive CLI collection
            }

            void CollectExtendedSnapshot(
                const DeviceMonitorConfig& config,
                DeviceSnapshot& snapshot,
                uint64_t now_ms) override
            {
#ifdef __APPLE__
                // Per-mountpoint disk usage via getmntinfo + statvfs
                snapshot.disks.clear();

                struct statfs* mounts = nullptr;
                int count = getmntinfo(&mounts, MNT_NOWAIT);
                if (count <= 0 || mounts == nullptr)
                    return;

                // Track the mount point with the largest capacity — this is the
                // main data volume. On modern macOS (Big Sur+) the root "/" is a
                // sealed read-only APFS snapshot; statvfs("/") may fail or return
                // f_blocks == 0. We fall back to the largest writable volume instead.
                uint64_t largest_total = 0;
                size_t best_idx = SIZE_MAX;

                std::set<uint64_t> seen_fsids;

                for (int i = 0; i < count; ++i)
                {
                    const std::string mnt_path(mounts[i].f_mntonname);
                    if (mnt_path.empty())
                        continue;

                    // Skip pseudo filesystems
                    const std::string fs_type(mounts[i].f_fstypename);
                    if (fs_type == "procfs" || fs_type == "devfs" ||
                        fs_type == "tmpfs" || fs_type == "ramfs" ||
                        fs_type == "fdescfs" || fs_type == "autofs" ||
                        fs_type == "sysfs" || fs_type == "fuse" ||
                        fs_type == "devtmpfs")
                        continue;

                    // On macOS APFS, /System/Volumes sub-volumes (VM, Preboot, Update, xarts,
                    // iSCPreboot, Hardware) share the same APFS container and are not meaningful
                    // for disk monitoring. Keep only /System/Volumes/Data (the writable volume).
                    if (mnt_path.find("/System/Volumes/") == 0 &&
                        mnt_path != "/System/Volumes/Data")
                        continue;

                    // Deduplicate by filesystem ID: APFS volumes in the same container share
                    // the same fsid. Keep only the first one encountered.
                    const auto& fsid = mounts[i].f_fsid;
                    uint64_t fsid_key = (static_cast<uint64_t>(static_cast<uint32_t>(fsid.val[0])) << 32) |
                                         static_cast<uint64_t>(static_cast<uint32_t>(fsid.val[1]));
                    if (!seen_fsids.insert(fsid_key).second)
                        continue;

                    // statvfs for actual usage
                    struct statvfs vfs;
                    if (statvfs(mnt_path.c_str(), &vfs) != 0)
                        continue;
                    if (vfs.f_blocks == 0)
                        continue;

                    uint64_t total_bytes = static_cast<uint64_t>(vfs.f_blocks) * vfs.f_frsize;
                    uint64_t used_bytes = (static_cast<uint64_t>(vfs.f_blocks) - static_cast<uint64_t>(vfs.f_bavail)) * vfs.f_frsize;

                    DiskInfo disk;
                    disk.path = mnt_path;
                    disk.total_bytes = total_bytes;
                    disk.used_bytes = used_bytes;
                    snapshot.disks.push_back(disk);

                    // Track the mount point with the largest total (main data volume)
                    if (total_bytes > largest_total)
                    {
                        largest_total = total_bytes;
                        best_idx = snapshot.disks.size() - 1;
                    }
                }

                // If the root "/" from CollectStaticInfo/CollectDynamicMetrics had
                // no usable data (statvfs("/") failed on the sealed system volume),
                // overwrite storage_usage/total_storage with the largest data volume.
                if (best_idx < snapshot.disks.size())
                {
                    const DiskInfo& best = snapshot.disks[best_idx];
                    if (!snapshot.metrics.storage_usage.available ||
                        snapshot.info.total_storage == 0)
                    {
                        double pct = (best.total_bytes > 0)
                            ? (static_cast<double>(best.used_bytes) / best.total_bytes) * 100.0
                            : 0.0;

                        snapshot.metrics.storage_usage.available = true;
                        snapshot.metrics.storage_usage.stale = false;
                        snapshot.metrics.storage_usage.value = pct;
                        snapshot.metrics.storage_usage.unit = "percent";
                        snapshot.metrics.storage_usage.source = "getmntinfo+statvfs(" + best.path + ")";
                        snapshot.metrics.storage_usage.collected_at_ms = now_ms;

                        snapshot.info.total_storage = best.total_bytes;
                    }
                }

                // Load average via sysctl
                {
                    double load[3] = {0, 0, 0};
                    size_t len = sizeof(load);
                    if (sysctlbyname("vm.loadavg", &load, &len, NULL, 0) == 0)
                    {
                        snapshot.metrics.load_average_1m.available = true;
                        snapshot.metrics.load_average_1m.value = load[0];
                        snapshot.metrics.load_average_1m.unit = "load";
                        snapshot.metrics.load_average_1m.source = "sysctl vm.loadavg (1m)";
                        snapshot.metrics.load_average_1m.collected_at_ms = now_ms;

                        snapshot.metrics.load_average_5m.available = true;
                        snapshot.metrics.load_average_5m.value = load[1];
                        snapshot.metrics.load_average_5m.unit = "load";
                        snapshot.metrics.load_average_5m.source = "sysctl vm.loadavg (5m)";
                        snapshot.metrics.load_average_5m.collected_at_ms = now_ms;

                        snapshot.metrics.load_average_15m.available = true;
                        snapshot.metrics.load_average_15m.value = load[2];
                        snapshot.metrics.load_average_15m.unit = "load";
                        snapshot.metrics.load_average_15m.source = "sysctl vm.loadavg (15m)";
                        snapshot.metrics.load_average_15m.collected_at_ms = now_ms;
                    }
                }
#endif
            }

        private:
            bool has_last_cpu_ = false;
            uint64_t last_cpu_total_ = 0;
            uint64_t last_cpu_used_ = 0;
        };

        std::shared_ptr<IDeviceProbe> CreateMacOSProbe()
        {
            return std::make_shared<MacOSProbe>();
        }
    }
}
