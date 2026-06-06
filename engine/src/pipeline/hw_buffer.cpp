// HwBuffer 实现
#include "pipeline/hw_buffer.h"
#include <unistd.h>
#include <iostream>

namespace aivision
{
    namespace pipeline
    {

        HwBuffer::HwBuffer(HwBufferDesc desc)
            : desc_(desc) {}

        HwBuffer::~HwBuffer() { Release(); }

        HwBuffer::HwBuffer(HwBuffer &&other) noexcept
            : desc_(other.desc_), release_cb_(std::move(other.release_cb_))
        {
            other.desc_.dma_fd = -1;
            other.desc_.dma_buf_fd = -1;
        }

        HwBuffer &HwBuffer::operator=(HwBuffer &&other) noexcept
        {
            if (this != &other)
            {
                Release();
                desc_ = other.desc_;
                release_cb_ = std::move(other.release_cb_);
                other.desc_.dma_fd = -1;
                other.desc_.dma_buf_fd = -1;
            }
            return *this;
        }

        void HwBuffer::Release()
        {
            if (desc_.dma_fd >= 0)
            {
                if (release_cb_)
                {
                    release_cb_(desc_);
                }
                else
                {
                    close(desc_.dma_fd);
                }
                desc_.dma_fd = -1;
            }
            if (desc_.dma_buf_fd >= 0)
            {
                close(desc_.dma_buf_fd);
                desc_.dma_buf_fd = -1;
            }
            desc_.size = 0;
        }

        HwBufferPtr MakeHwBuffer(HwBufferDesc desc)
        {
            return std::make_shared<HwBuffer>(desc);
        }

        // HwBufferPool 实现
        struct HwBufferPool::Impl
        {
            std::mutex mutex;
            std::unordered_map<int, HwBufferPtr> buffers;
        };

        HwBufferPool::HwBufferPool(size_t pool_size)
            : total_bytes_(pool_size), impl_(std::make_unique<Impl>()) {}

        HwBufferPool::~HwBufferPool() = default;

        void HwBufferPool::Register(HwBufferPtr buf)
        {
            std::lock_guard<std::mutex> lock(impl_->mutex);
            if (buf && buf->IsValid())
            {
                impl_->buffers[buf->Desc().dma_fd] = buf;
                used_bytes_.fetch_add(buf->Desc().size);
            }
        }

        void HwBufferPool::Reclaim(int dma_fd)
        {
            std::lock_guard<std::mutex> lock(impl_->mutex);
            auto it = impl_->buffers.find(dma_fd);
            if (it != impl_->buffers.end())
            {
                used_bytes_.fetch_sub(it->second->Desc().size);
                impl_->buffers.erase(it);
            }
        }

    } // namespace pipeline
} // namespace aivision
