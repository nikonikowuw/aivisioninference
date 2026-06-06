// AlgoInstance 实现
#include "algo/algo_instance.h"
#include <iostream>

namespace aivision
{
    namespace algo
    {

        AlgoInstance::AlgoInstance(std::shared_ptr<SoHandle> so_handle,
                                   const std::string &algo_name,
                                   const std::string &version)
            : so_handle_(std::move(so_handle)), algo_name_(algo_name), version_(version) {}

        AlgoInstance::~AlgoInstance()
        {
            if (algo_handle_)
            {
                DestroyInternal();
            }
        }

        bool AlgoInstance::Initialize(const std::string &config_json)
        {
            if (!so_handle_ || !so_handle_->IsLoaded())
                return false;

            algo_handle_ = so_handle_->Init(config_json.c_str());
            if (!algo_handle_)
            {
                state_.store(AlgoInstanceState::Error);
                return false;
            }

            state_.store(AlgoInstanceState::Ready);
            return true;
        }

        bool AlgoInstance::Infer(const pipeline::HwBufferDesc &input_desc,
                                 const std::string &context_json,
                                 std::string &result_json,
                                 uint32_t &infer_time_us)
        {
            if (!algo_handle_ || !so_handle_)
                return false;

            state_.store(AlgoInstanceState::Running);

            hw_buffer_desc_t fb_desc;
            fb_desc.dma_fd = input_desc.dma_fd;
            fb_desc.size = input_desc.size;
            fb_desc.width = input_desc.width;
            fb_desc.height = input_desc.height;
            fb_desc.pixel_format = input_desc.pixel_format;
            fb_desc.dma_buf_fd = input_desc.dma_buf_fd;
            fb_desc.phys_addr = input_desc.phys_addr;

            infer_result_t result;
            int ret = so_handle_->Infer(
                algo_handle_,
                &fb_desc,
                context_json.empty() ? nullptr : context_json.c_str(),
                &result);

            if (ret == 0 && result.result_json)
            {
                result_json = result.result_json;
                infer_time_us = result.infer_time_us;
                algo_free_result(&result);
            }

            state_.store(AlgoInstanceState::Ready);
            return ret == 0;
        }

        bool AlgoInstance::SelfTest()
        {
            if (!so_handle_)
                return false;

            state_.store(AlgoInstanceState::SelfTesting);
            int ret = so_handle_->SelfTest();
            state_.store(ret == 0 ? AlgoInstanceState::Ready
                                  : AlgoInstanceState::Error);
            return ret == 0;
        }

        void AlgoInstance::DestroyInternal()
        {
            if (so_handle_ && algo_handle_)
            {
                so_handle_->Destroy(algo_handle_);
                algo_handle_ = nullptr;
            }
            state_.store(AlgoInstanceState::Unloading);
        }

    } // namespace algo
} // namespace aivision
