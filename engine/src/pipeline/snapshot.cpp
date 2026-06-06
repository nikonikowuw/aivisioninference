// Snapshot 实现
#include "pipeline/snapshot.h"

namespace aivision
{
    namespace pipeline
    {

        void SnapshotManager::UpdateConfig(const std::string &algo_name,
                                           const AlgoConfig &config)
        {
            std::unique_lock<std::shared_mutex> lock(rw_mutex_);
            auto &existing = configs_[algo_name];
            existing = config;
            existing.version++;
        }

        void SnapshotManager::UpdateAllConfigs(
            const std::unordered_map<std::string, AlgoConfig> &configs)
        {
            std::unique_lock<std::shared_mutex> lock(rw_mutex_);
            for (const auto &[name, config] : configs)
            {
                auto &existing = configs_[name];
                existing = config;
                existing.version++;
            }
        }

        AlgoConfigSnapshot SnapshotManager::GetSnapshot(const std::string &algo_name)
        {
            std::shared_lock<std::shared_mutex> lock(rw_mutex_);
            auto it = configs_.find(algo_name);
            if (it != configs_.end())
            {
                return it->second; // 返回深度拷贝 (字符串拷贝)
            }
            return AlgoConfigSnapshot{};
        }

        void SnapshotManager::RemoveConfig(const std::string &algo_name)
        {
            std::unique_lock<std::shared_mutex> lock(rw_mutex_);
            configs_.erase(algo_name);
        }

        uint64_t SnapshotManager::GetVersion(const std::string &algo_name)
        {
            std::shared_lock<std::shared_mutex> lock(rw_mutex_);
            auto it = configs_.find(algo_name);
            return it != configs_.end() ? it->second.version : 0;
        }

        void SnapshotManager::Clear()
        {
            std::unique_lock<std::shared_mutex> lock(rw_mutex_);
            configs_.clear();
        }

    } // namespace pipeline
} // namespace aivision
