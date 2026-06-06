#ifndef AIVISION_PIPELINE_HAL_H
#define AIVISION_PIPELINE_HAL_H

// HAL (Hardware Abstraction Layer) — 硬件加速流水线接口。
// 核心设计：
//   1. 定义统一的拉流、解码、硬件缩放/Resize 接口。
//   2. 各平台 (RKMPP, FFmpeg, Ascend) 编译为独立动态库。
//   3. 引擎主程序通过配置文件动态加载指定平台的 .so。
//   4. 使用智能指针管理动态库生命周期 (RAII)。

#include <cstdint>
#include <functional>
#include <memory>
#include <string>

#include "hw_buffer.h"

namespace aivision
{
    namespace pipeline
    {

        /// 媒体帧回调
        using FrameCallback = std::function<void(HwBufferPtr frame)>;

        /// 流水线状态回调
        enum class PipelineState
        {
            Idle,
            Connecting,
            Streaming,
            Paused,
            Error,
            Stopped,
        };

        using StateCallback = std::function<void(PipelineState state)>;

        /// 硬件加速流水线接口
        class IMediaPipeline
        {
        public:
            virtual ~IMediaPipeline() = default;

            /// 初始化流水线
            virtual bool Initialize(const std::string &config_json) = 0;

            /// 开始拉流
            virtual bool Start(const std::string &url) = 0;

            /// 停止拉流
            virtual void Stop() = 0;

            /// 暂停
            virtual void Pause() = 0;

            /// 恢复
            virtual void Resume() = 0;

            /// 是否正在运行
            virtual bool IsRunning() const = 0;

            /// 设置帧回调
            virtual void SetFrameCallback(FrameCallback cb) = 0;

            /// 设置状态回调
            virtual void SetStateCallback(StateCallback cb) = 0;

            /// 获取流水线类型名称
            virtual std::string GetPipelineType() const = 0;

            /// 获取当前状态
            virtual PipelineState GetState() const = 0;

            // ============================================================
            // 编码器接口 (可选实现)
            // ============================================================

            /// 初始化编码器
            virtual bool EncodeInit(const std::string &config_json) { return false; }

            /// 编码一帧。输入 YUV (DMA Buffer)，输出 H.264/H.265 NALU 包。
            /// data 为输出缓冲区，size 为输入缓冲区大小，out_size 为实际编码后大小。
            virtual bool EncodeFrame(HwBufferPtr frame, uint8_t *data, size_t size, size_t &out_size) { return false; }

            /// 销毁编码器
            virtual void EncodeDestroy() {}

            /// 解码器类型
            virtual int GetDecodeHWType() const = 0;
        };

        /// 流水线工厂函数类型 (每个动态库导出此函数)
        using CreatePipelineFunc = IMediaPipeline *(*)();
        using DestroyPipelineFunc = void (*)(IMediaPipeline *);

        /// HAL 管理器 — 动态加载平台流水线实现
        class HALManager
        {
        public:
            HALManager() = default;
            ~HALManager() = default;

            /// 加载指定平台的流水线动态库
            bool LoadPipeline(const std::string &so_path,
                              const std::string &config_json);

            /// 获取当前加载的流水线实例
            IMediaPipeline *GetPipeline() const { return pipeline_.get(); }

            /// 卸载当前流水线
            void Unload();

            /// 检查是否已加载
            bool IsLoaded() const { return pipeline_ != nullptr; }

            /// 获取已加载的平台名称
            std::string GetLoadedPlatform() const;

            /// 动态库路径
            std::string GetSoPath() const { return so_path_; }

        private:
            std::string so_path_;
            void *dl_handle_ = nullptr;
            std::unique_ptr<IMediaPipeline, DestroyPipelineFunc> pipeline_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_HAL_H
