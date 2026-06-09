// AlgoManager 实现
#include "algo/algo_manager.h"
#include <iostream>
#include <thread>
#include <chrono>

namespace aivision
{
    namespace algo
    {

        AlgoManager::~AlgoManager()
        {
            // 清理所有实例
            std::lock_guard<std::mutex> lock(mutex_);
            for (auto &[name, instance] : instances_)
            {
                (void)name;
                if (instance)
                    instance->Destroy();
            }
            instances_.clear();
        }

        AlgoInstancePtr AlgoManager::Load(const std::string &algo_name,
                                          const std::string &version,
                                          const std::string &so_path,
                                          const std::string &config_json)
        {
            // 检查资源
            if (!CanLoad(so_path))
            {
                std::cerr << "[AlgoManager] Cannot load " << so_path
                          << ": insufficient resources" << std::endl;
                return nullptr;
            }

            try
            {
                auto so_handle = std::make_shared<SoHandle>(so_path);
                auto instance = std::make_shared<AlgoInstance>(
                    so_handle, algo_name, version);

                if (!instance->Initialize(config_json))
                {
                    std::cerr << "[AlgoManager] Failed to initialize algo: "
                              << algo_name << std::endl;
                    return nullptr;
                }

                std::lock_guard<std::mutex> lock(mutex_);
                instances_[algo_name] = instance;
                return instance;
            }
            catch (const SoLoadException &e)
            {
                std::cerr << "[AlgoManager] SoLoadException: " << e.what()
                          << std::endl;
                return nullptr;
            }
        }

        bool AlgoManager::Unload(const std::string &algo_name)
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = instances_.find(algo_name);
            if (it == instances_.end())
                return false;

            it->second->Destroy();
            instances_.erase(it);
            return true;
        }

        std::pair<AlgoInstancePtr, bool> AlgoManager::Acquire(
            const std::string &algo_name, uint32_t timeout_ms)
        {
            std::unique_lock<std::mutex> lock(mutex_);

            // 如果正在热更新，阻塞等待
            if (hot_reload_state_.load() != HotReloadState::Idle)
            {
                if (!hot_reload_cv_.wait_for(
                        lock, std::chrono::milliseconds(timeout_ms),
                        [this]()
                        {
                            return hot_reload_state_.load() == HotReloadState::Idle;
                        }))
                {
                    return {nullptr, false}; // 超时
                }
            }

            auto it = instances_.find(algo_name);
            if (it == instances_.end())
            {
                return {nullptr, false};
            }

            it->second->AddRef();
            return {it->second, true};
        }

        void AlgoManager::Release(const std::string &algo_name)
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = instances_.find(algo_name);
            if (it != instances_.end())
            {
                it->second->Release();
            }
        }

        HotReloadResult AlgoManager::HotReload(const std::string &algo_name,
                                               const std::string &new_version,
                                               const std::string &new_so_path,
                                               const std::string &new_config_json,
                                               uint32_t drain_timeout_ms)
        {
            HotReloadResult result;
            auto prev_state = hot_reload_state_.load();

            // 状态机: IDLE → PAUSING
            if (!hot_reload_state_.compare_exchange_strong(
                    prev_state, HotReloadState::Pausing))
            {
                result.success = false;
                result.error_message = "Hot reload already in progress";
                return result;
            }
            NotifyStateChange(prev_state, HotReloadState::Pausing);
            prev_state = HotReloadState::Pausing;

            // PAUSING: 暂停 Worker 派发
            // (外部调用者负责调用 WorkerPool::PauseAll())
            hot_reload_state_.store(HotReloadState::Draining);
            NotifyStateChange(prev_state, HotReloadState::Draining);

            // DRAINING: 等待引用计数归零
            auto drain_start = std::chrono::steady_clock::now();
            {
                std::unique_lock<std::mutex> lock(mutex_);
                auto it = instances_.find(algo_name);
                if (it != instances_.end())
                {
                    while (it->second->GetRefCount() > 0)
                    {
                        if (std::chrono::duration_cast<std::chrono::milliseconds>(
                                std::chrono::steady_clock::now() - drain_start)
                                .count() >= drain_timeout_ms)
                        {
                            drain_timed_out_.store(true);
                            std::cerr << "[AlgoManager] Drain timeout for "
                                      << algo_name << std::endl;
                            break;
                        }
                        std::this_thread::sleep_for(std::chrono::milliseconds(10));
                    }
                }
            }
            prev_state = HotReloadState::Draining;

            // DRAINING → LOADING_NEW
            hot_reload_state_.store(HotReloadState::LoadingNew);
            NotifyStateChange(prev_state, HotReloadState::LoadingNew);
            prev_state = HotReloadState::LoadingNew;

            auto load_start = std::chrono::steady_clock::now();

            // 先加载新版本 (成功后再卸载旧版本)
            AlgoInstancePtr new_instance;
            try
            {
                auto so_handle = std::make_shared<SoHandle>(new_so_path);
                new_instance = std::make_shared<AlgoInstance>(
                    so_handle, algo_name, new_version);

                if (!new_instance->Initialize(new_config_json))
                {
                    throw std::runtime_error("Initialization failed");
                }
            }
            catch (const std::exception &e)
            {
                result.error_message = "Failed to load new version: ";
                result.error_message += e.what();

                // ROLLING_BACK
                hot_reload_state_.store(HotReloadState::RollingBack);
                NotifyStateChange(prev_state, HotReloadState::RollingBack);

                // 回滚到 IDLE，旧版本仍在
                hot_reload_state_.store(HotReloadState::Idle);
                result.final_state = HotReloadState::RollingBack;
                return result;
            }

            auto load_end = std::chrono::steady_clock::now();
            result.load_time_ms = std::chrono::duration_cast<
                                      std::chrono::milliseconds>(load_end - load_start)
                                      .count();

            prev_state = HotReloadState::LoadingNew;

            // LOADING_NEW → SELF_TESTING
            hot_reload_state_.store(HotReloadState::SelfTesting);
            NotifyStateChange(prev_state, HotReloadState::SelfTesting);
            prev_state = HotReloadState::SelfTesting;

            // 自检
            bool selftest_ok = new_instance->SelfTest();
            result.self_check_result = selftest_ok ? "passed" : "failed";

            if (!selftest_ok)
            {
                result.error_message = "Self-test failed for new version";

                hot_reload_state_.store(HotReloadState::RollingBack);
                NotifyStateChange(prev_state, HotReloadState::RollingBack);

                hot_reload_state_.store(HotReloadState::Idle);
                result.final_state = HotReloadState::RollingBack;
                return result;
            }

            // SELF_TESTING → RESUMING
            hot_reload_state_.store(HotReloadState::Resuming);
            NotifyStateChange(prev_state, HotReloadState::Resuming);

            // 原子替换: 新实例替换旧实例
            if (!AtomicSwap(algo_name, new_instance))
            {
                result.error_message = "Atomic swap failed";
                hot_reload_state_.store(HotReloadState::RollingBack);
                hot_reload_state_.store(HotReloadState::Idle);
                result.final_state = HotReloadState::RollingBack;
                return result;
            }

            prev_state = HotReloadState::Resuming;

            // RESUMING → IDLE
            hot_reload_state_.store(HotReloadState::Idle);
            NotifyStateChange(prev_state, HotReloadState::Idle);

            // 通知等待中的 Acquire
            hot_reload_cv_.notify_all();

            result.success = true;
            result.final_state = HotReloadState::Idle;
            return result;
        }

        bool AlgoManager::AtomicSwap(const std::string &algo_name,
                                     AlgoInstancePtr new_instance)
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = instances_.find(algo_name);
            if (it != instances_.end())
            {
                // 旧版本会在 shared_ptr 引用归零时自动 dlclose
                instances_.erase(it);
            }
            instances_[algo_name] = std::move(new_instance);
            return true;
        }

        bool AlgoManager::CanLoad(const std::string &so_path)
        {
            (void)so_path;
            // TODO: 检查 NPU 内存是否充足
            return true;
        }

        std::vector<AlgoInstancePtr> AlgoManager::GetAllInstances() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            std::vector<AlgoInstancePtr> result;
            result.reserve(instances_.size());
            for (const auto &[_, instance] : instances_)
            {
                result.push_back(instance);
            }
            return result;
        }

        size_t AlgoManager::InstanceCount() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            return instances_.size();
        }

    } // namespace algo
} // namespace aivision
