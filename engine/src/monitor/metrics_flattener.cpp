#include "monitor/metrics_flattener.h"
#include <cmath>

namespace aivision
{
    namespace monitor
    {

        MetricsFlattener::MetricsFlattener()
        {
        }

        FlattenedMetrics MetricsFlattener::Flatten(const DeviceSnapshot& snapshot, uint64_t interval_ms)
        {
            FlattenedMetrics m;

            if (snapshot.timestamp_ms == 0)
                return m;

            // --- CPU ---
            m.cpu_usage = snapshot.metrics.cpu_usage.available
                              ? snapshot.metrics.cpu_usage.value
                              : 0.0;

            // --- Load Average (from /proc/loadavg or diagnostics if available) ---
            // The LinuxGenericProbe might store load in diagnostics; try to find it
            for (const auto& diag : snapshot.diagnostics)
            {
                if (diag.name.find("linux") != std::string::npos ||
                    diag.name.find("generic") != std::string::npos)
                {
                    for (const auto& ev : diag.evidence)
                    {
                        if (ev.find("load_1m:") == 0)
                            m.cpu_load_1m = std::stod(ev.substr(8));
                        if (ev.find("load_5m:") == 0)
                            m.cpu_load_5m = std::stod(ev.substr(8));
                        if (ev.find("load_15m:") == 0)
                            m.cpu_load_15m = std::stod(ev.substr(9));
                    }
                }
            }

            // --- Memory ---
            m.memory_usage = snapshot.metrics.memory_usage.available
                                 ? snapshot.metrics.memory_usage.value
                                 : 0.0;
            m.memory_total = snapshot.info.total_memory;
            // Estimate memory_used from memory_usage percentage
            if (m.memory_total > 0 && m.memory_usage > 0)
            {
                m.memory_used = static_cast<uint64_t>(
                    (m.memory_usage / 100.0) * m.memory_total);
            }

            // --- Disk ---
            m.disk_usage = ExtractDiskUsage(snapshot);

            // --- Temperature ---
            m.temperature = snapshot.metrics.temperature.available
                                ? snapshot.metrics.temperature.value
                                : 0.0;

            // --- Uptime ---
            // Uptime is typically stored in snapshot.info as total_storage or
            // may need to be read separately. We estimate from timestamp.
            // (Actual uptime comes from /proc/uptime in the heartbeat reporter)
            m.uptime = 0; // Will be set by HeartbeatReporter

            // --- Process/Thread counts ---
            // These are currently not part of DeviceDynamicMetrics,
            // so we leave them as 0 for now. The heartbeat reporter
            // may read them directly.

            // --- Network: compute speed from delta ---
            // Attempt to extract network bytes from probe diagnostics
            for (const auto& diag : snapshot.diagnostics)
            {
                for (const auto& ev : diag.evidence)
                {
                    if (ev.find("net_rx_bytes:") == 0)
                        m.net_rx_bytes = std::stoull(ev.substr(13));
                    if (ev.find("net_tx_bytes:") == 0)
                        m.net_tx_bytes = std::stoull(ev.substr(13));
                }
            }

            // Calculate network speed (bytes per second)
            if (last_timestamp_ms_ > 0 && interval_ms > 0)
            {
                if (m.net_rx_bytes >= last_net_rx_bytes_)
                {
                    uint64_t rx_delta = m.net_rx_bytes - last_net_rx_bytes_;
                    m.net_rx_speed = (static_cast<double>(rx_delta) * 1000.0) /
                                     static_cast<double>(interval_ms);
                }
                if (m.net_tx_bytes >= last_net_tx_bytes_)
                {
                    uint64_t tx_delta = m.net_tx_bytes - last_net_tx_bytes_;
                    m.net_tx_speed = (static_cast<double>(tx_delta) * 1000.0) /
                                     static_cast<double>(interval_ms);
                }
            }

            // Store for next delta computation
            last_net_rx_bytes_ = m.net_rx_bytes;
            last_net_tx_bytes_ = m.net_tx_bytes;
            last_timestamp_ms_ = snapshot.timestamp_ms;

            return m;
        }

        std::vector<DiskUsageInfo> MetricsFlattener::ExtractDiskUsage(const DeviceSnapshot& snapshot)
        {
            std::vector<DiskUsageInfo> disks;

            // Disk usage may be in diagnostics or in storage_usage
            // The LinuxGenericProbe's expensive probe populates storage_usage
            DiskUsageInfo root;
            root.path = "/";
            if (snapshot.metrics.storage_usage.available)
            {
                root.percent = snapshot.metrics.storage_usage.value;
                if (snapshot.info.total_storage > 0)
                {
                    root.total = snapshot.info.total_storage;
                    root.used = static_cast<uint64_t>(
                        (root.percent / 100.0) * root.total);
                }
            }
            disks.push_back(root);

            // Try to find additional mount point details in diagnostics
            for (const auto& diag : snapshot.diagnostics)
            {
                for (const auto& ev : diag.evidence)
                {
                    if (ev.find("disk:") == 0)
                    {
                        // Format: "disk:/path:total:used:percent"
                        auto parts = ev;
                        DiskUsageInfo di;
                        size_t pos1 = parts.find(':');
                        if (pos1 == std::string::npos) continue;
                        size_t pos2 = parts.find(':', pos1 + 1);
                        if (pos2 == std::string::npos) continue;
                        size_t pos3 = parts.find(':', pos2 + 1);
                        if (pos3 == std::string::npos) continue;
                        di.path = parts.substr(pos1 + 1, pos2 - pos1 - 1);
                        di.total = std::stoull(parts.substr(pos2 + 1, pos3 - pos2 - 1));
                        di.used = std::stoull(parts.substr(pos3 + 1));
                        if (di.total > 0)
                            di.percent = (static_cast<double>(di.used) / di.total) * 100.0;
                        // Replace root if already present
                        bool found = false;
                        for (auto& d : disks)
                        {
                            if (d.path == di.path)
                            {
                                d = di;
                                found = true;
                                break;
                            }
                        }
                        if (!found)
                            disks.push_back(di);
                    }
                }
            }

            return disks;
        }

    } // namespace monitor
} // namespace aivision
