#ifndef AIVISION_PIPELINE_SNAPSHOT_H
#define AIVISION_PIPELINE_SNAPSHOT_H

// Snapshot — 读写锁快照机制。
// 核心设计：
//   1. Go 控制面在运行中下发 ROI/MARK/LINE 配置变更。
//   2. C++ Worker 从队列取出 FrameContext 准备推理前，
//      获取读锁 (shared_lock)，将当前 AlgoConfig 快照拷贝一份。
//   3. 立即解锁，后续慢速推理周期使用快照配置，不受后续配置变更影响。
//   4. 写锁由配置更新线程持有，确保配置原子性更新。

#include <atomic>
#include <mutex>
#include <shared_mutex>
#include <string>

namespace aivision
{
    namespace pipeline
    {

        /// 算法配置 (可被快照)
        struct AlgoConfig
        {
            /// 算法名称
            std::string algo_name;

            /// 算法版本
            std::string algo_version;

            /// ROI 区域 (JSON 字符串)
            std::string roi_regions_json;

            /// MARK 屏蔽区域 (JSON 字符串)
            std::string mark_regions_json;

            /// LINE 绊线区域 (JSON 字符串)
            std::string line_regions_json;

            /// 算法特定参数 (JSON 透传)
            std::string algo_params_json;

            /// 版本号 (每次写操作递增，用于检测变更)
            uint64_t version = 0;

            /// 检查配置是否为空
            bool IsEmpty() const { return algo_name.empty(); }
        };

        /// 配置快照 (Worker 实际使用)
        using AlgoConfigSnapshot = AlgoConfig;

        /// 快照管理器 — 管理单个流的所有算法配置
        class SnapshotManager
        {
        public:
            SnapshotManager() = default;
            ~SnapshotManager() = default;

            /// 更新配置 (写锁，由 Go 配置下发线程调用)
            void UpdateConfig(const std::string &algo_name, const AlgoConfig &config);

            /// 批量更新所有算法配置
            void UpdateAllConfigs(
                const std::unordered_map<std::string, AlgoConfig> &configs);

            /// 获取配置快照 (读锁，由 Worker 推理前调用)
            /// 返回当前配置的深度拷贝。
            AlgoConfigSnapshot GetSnapshot(const std::string &algo_name);

            /// 移除指定算法的配置
            void RemoveConfig(const std::string &algo_name);

            /// 获取当前配置版本号
            uint64_t GetVersion(const std::string &algo_name);

            /// 清空所有配置
            void Clear();

        private:
            /// 读写锁
            mutable std::shared_mutex rw_mutex_;

            /// 存储所有算法配置 (algo_name -> AlgoConfig)
            std::unordered_map<std::string, AlgoConfig> configs_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_SNAPSHOT_H
