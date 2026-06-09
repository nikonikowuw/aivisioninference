#ifndef AIVISION_ALGO_ALGO_INSTANCE_H
#define AIVISION_ALGO_ALGO_INSTANCE_H

// AlgoInstance — 封装加载后的算法实例。
// 核心设计：
//   1. 持有 SoHandle 和算法上下文句柄 (algo_handle_t)。
//   2. atomic 引用计数 (AddRef/Release) 确保热更新时安全释放。
//   3. Infer() 方法封装推理调用。
//   4. SelfTest() 方法执行自检。

#include <atomic>
#include <chrono>
#include <memory>
#include <string>

#include "so_handle.h"
#include "pipeline/hw_buffer.h"

namespace aivision
{
    namespace algo
    {

        /// AlgoInstance 状态
        enum class AlgoInstanceState
        {
            Loading,
            Ready,
            Running,
            SelfTesting,
            Error,
            Unloading,
        };

        /// AlgoInstance — 算法实例
        class AlgoInstance
        {
        public:
            AlgoInstance(std::shared_ptr<SoHandle> so_handle,
                         const std::string &algo_name,
                         const std::string &version);

            ~AlgoInstance();

            /// 初始化 (调用 detector_init)
            bool Initialize(const std::string &config_json);

            /// 推理 (调用 detector_infer)
            bool Infer(const pipeline::HwBufferDesc &input_desc,
                       const std::string &context_json,
                       std::string &result_json,
                       uint32_t &infer_time_us);

            /// 自检 (调用 detector_self_test)
            bool SelfTest();

            /// 增加引用计数
            int AddRef() { return ref_count_.fetch_add(1) + 1; }

            /// 减少 Worker 使用引用计数；销毁由 Destroy() 负责。
            int Release()
            {
                int expected = ref_count_.load();
                while (expected > 0)
                {
                    if (ref_count_.compare_exchange_weak(expected, expected - 1))
                    {
                        return expected - 1;
                    }
                }
                return expected;
            }

            /// 获取当前引用计数
            int GetRefCount() const { return ref_count_.load(); }

            /// 获取算法名称
            const std::string &GetName() const { return algo_name_; }

            /// 获取版本
            const std::string &GetVersion() const { return version_; }

            /// 获取状态
            AlgoInstanceState GetState() const { return state_.load(); }

            /// 检查是否可用 (Ready 状态)
            bool IsReady() const { return state_.load() == AlgoInstanceState::Ready; }

            /// 获取 SoHandle 引用
            std::shared_ptr<SoHandle> GetSoHandle() const { return so_handle_; }

            /// 显式销毁算法上下文，由 AlgoManager 卸载/析构时调用。
            void Destroy() { DestroyInternal(); }

        private:
            /// 内部销毁
            void DestroyInternal();

            std::shared_ptr<SoHandle> so_handle_;
            std::string algo_name_;
            std::string version_;
            std::atomic<AlgoInstanceState> state_{AlgoInstanceState::Loading};
            std::atomic<int> ref_count_{0};

            /// 算法上下文句柄 (由 detector_init 返回)
            algo_handle_t algo_handle_{nullptr};
        };

        /// AlgoInstance 的智能指针
        using AlgoInstancePtr = std::shared_ptr<AlgoInstance>;

    } // namespace algo
} // namespace aivision

#endif // AIVISION_ALGO_ALGO_INSTANCE_H
