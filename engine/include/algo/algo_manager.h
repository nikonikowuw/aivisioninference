#ifndef AIVISION_ALGO_ALGO_MANAGER_H
#define AIVISION_ALGO_ALGO_MANAGER_H

// AlgoManager — 管理所有算法实例的生命周期。
// 核心功能：
//   1. Load — 加载算法 .so 并创建 AlgoInstance。
//   2. Unload — 卸载指定算法。
//   3. Acquire — Worker 获取算法实例 (热更新时阻塞等待)。
//   4. HotReload — 热更新状态机。
//   5. Release — Worker 释放实例引用。

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>

#include "algo_instance.h"

namespace aivision
{
    namespace algo
    {

        /// 热更新状态枚举 (对应 HotReload 状态机)
        enum class HotReloadState
        {
            Idle,
            Pausing,
            Draining,
            LoadingNew,
            SelfTesting,
            Resuming,
            RollingBack,
        };

        /// 热更新结果
        struct HotReloadResult
        {
            bool success = false;
            HotReloadState final_state = HotReloadState::Idle;
            std::string error_message;
            uint64_t load_time_ms = 0;
            uint64_t npu_memory_bytes = 0;
            std::string self_check_result;
        };

        /// AlgoManager — 算法管理器
        class AlgoManager
        {
        public:
            AlgoManager() = default;
            ~AlgoManager();

            /// 加载一个算法实例
            AlgoInstancePtr Load(const std::string &algo_name,
                                 const std::string &version,
                                 const std::string &so_path,
                                 const std::string &config_json);

            /// 卸载指定算法
            bool Unload(const std::string &algo_name);

            /// 获取算法实例 (Worker 调用)
            /// 热更新期间会阻塞直到新版本就绪。
            /// timeout_ms 为最大等待时间。
            std::pair<AlgoInstancePtr, bool> Acquire(const std::string &algo_name,
                                                     uint32_t timeout_ms = 30000);

            /// 释放算法实例 (Worker 调用)
            void Release(const std::string &algo_name);

            /// 热更新算法
            /// @param algo_name 算法名称
            /// @param new_version 新版本号
            /// @param new_so_path 新 .so 路径
            /// @param new_config_json 新配置参数
            /// @param drain_timeout_ms 排空超时 (默认 30000ms)
            HotReloadResult HotReload(const std::string &algo_name,
                                      const std::string &new_version,
                                      const std::string &new_so_path,
                                      const std::string &new_config_json,
                                      uint32_t drain_timeout_ms = 30000);

            /// 获取热更新状态
            HotReloadState GetHotReloadState() const { return hot_reload_state_.load(); }

            /// 检查是否有正在执行的热更新
            bool IsHotReloading() const
            {
                return hot_reload_state_.load() != HotReloadState::Idle;
            }

            /// 获取当前所有加载的算法实例
            std::vector<AlgoInstancePtr> GetAllInstances() const;

            /// 获取算法实例数
            size_t InstanceCount() const;

            /// 设置热更新状态变更回调
            using StateChangeCallback = std::function<void(HotReloadState, HotReloadState)>;
            void SetStateChangeCallback(StateChangeCallback cb)
            {
                state_change_cb_ = std::move(cb);
            }

        private:
            /// 内部: 从旧版本切换到新版本 (原子替换)
            bool AtomicSwap(const std::string &algo_name,
                            AlgoInstancePtr new_instance);

            /// 通知状态变更
            void NotifyStateChange(HotReloadState from, HotReloadState to)
            {
                if (state_change_cb_)
                {
                    state_change_cb_(from, to);
                }
            }

            /// 检查实例是否可加载 (内存/硬件资源)
            bool CanLoad(const std::string &so_path);

            /// 互斥锁保护实例映射
            mutable std::mutex mutex_;

            /// 算法实例映射 (algo_name -> AlgoInstancePtr)
            std::unordered_map<std::string, AlgoInstancePtr> instances_;

            /// 热更新状态
            std::atomic<HotReloadState> hot_reload_state_{HotReloadState::Idle};

            /// 热更新等待条件变量 (Acquire 在此等待)
            std::condition_variable hot_reload_cv_;

            /// 状态变更回调
            StateChangeCallback state_change_cb_;

            /// 排空超时标志
            std::atomic<bool> drain_timed_out_{false};
        };

    } // namespace algo
} // namespace aivision

#endif // AIVISION_ALGO_ALGO_MANAGER_H
