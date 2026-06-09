#ifndef AIVISION_PIPELINE_HW_BUFFER_H
#define AIVISION_PIPELINE_HW_BUFFER_H

// HwBuffer — 物理 DMA 内存包装器，支持零拷贝流转。
// 核心设计：
//   1. 包装解码器输出的物理 DMA 内存块。
//   2. 使用自定义删除器的 shared_ptr 实现引用计数自动回收。
//   3. HwBufferDesc 结构体包含 dma_fd，可在进程内传递（不跨进程）。

#include <cstdint>
#include <functional>
#include <memory>

namespace aivision
{
    namespace pipeline
    {

        enum class HwBufferMemoryType
        {
            Unknown = 0,
            DMABuf,
            CVPixelBuffer,
            HostMemory,
        };

        /// 硬件缓冲区描述符 — 用于在解码器、NPU、算法 .so 之间零拷贝传递。
        /// 关键约束：平台原生句柄仅在同一进程内有效，不跨进程传递。
        struct HwBufferDesc
        {
            /// 内存类型
            HwBufferMemoryType memory_type = HwBufferMemoryType::Unknown;

            /// DMA 文件描述符 (来自 V4L2 / DRM 等)
            int dma_fd = -1;

            /// 缓冲区大小 (字节)
            size_t size = 0;

            /// 图像宽度 (像素)
            uint32_t width = 0;

            /// 图像高度 (像素)
            uint32_t height = 0;

            /// 像素格式 (如 V4L2_PIX_FMT_NV12)
            uint32_t pixel_format = 0;

            /// DMA 缓冲区文件描述符 (用于 RGA 等硬件缩放/旋转)
            int dma_buf_fd = -1;

            /// 物理地址 (某些 NPU 需要)
            uint64_t phys_addr = 0;

            /// 平台原生缓冲句柄，例如 macOS CVPixelBufferRef/IOSurfaceRef
            void *native_handle = nullptr;
        };

        /// HwBuffer — RAII 包装 DMA 缓冲区
        class HwBuffer
        {
        public:
            /// 默认构造（空缓冲区）
            HwBuffer() = default;

            /// 从描述符构造，注册自定义删除器
            explicit HwBuffer(HwBufferDesc desc);

            /// 析构时自动释放 DMA 内存 (通过删除器回调)
            ~HwBuffer();

            /// 禁用拷贝，允许移动
            HwBuffer(const HwBuffer &) = delete;
            HwBuffer &operator=(const HwBuffer &) = delete;
            HwBuffer(HwBuffer &&other) noexcept;
            HwBuffer &operator=(HwBuffer &&other) noexcept;

            /// 获取描述符引用
            const HwBufferDesc &Desc() const { return desc_; }

            /// 检查是否有效
            bool IsValid() const { return desc_.dma_fd >= 0 || desc_.native_handle != nullptr; }

            /// 释放底层资源 (手动触发)
            void Release();

            /// 设置自定义释放回调
            using ReleaseCallback = std::function<void(const HwBufferDesc &)>;
            void SetReleaseCallback(ReleaseCallback cb) { release_cb_ = std::move(cb); }

        private:
            HwBufferDesc desc_{};
            ReleaseCallback release_cb_;
        };

        /// HwBuffer 的智能指针别名 (引用计数自动回收)
        using HwBufferPtr = std::shared_ptr<HwBuffer>;

        /// 创建 HwBuffer 的智能指针
        HwBufferPtr MakeHwBuffer(HwBufferDesc desc);

        /// 硬件缓冲池管理器
        class HwBufferPool
        {
        public:
            explicit HwBufferPool(size_t pool_size = 0);
            ~HwBufferPool();

            /// 向池中注册一个缓冲区
            void Register(HwBufferPtr buf);

            /// 从池中回收一个缓冲区
            void Reclaim(int dma_fd);

            /// 获取已用字节数
            uint64_t UsedBytes() const { return used_bytes_.load(); }

            /// 获取总容量
            uint64_t TotalBytes() const { return total_bytes_; }

        private:
            struct Impl;
            std::unique_ptr<Impl> impl_;
            std::atomic<uint64_t> used_bytes_{0};
            uint64_t total_bytes_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_HW_BUFFER_H
