// AlgoInstance 实现
#include "algo/algo_instance.h"
#include <iostream>
#include <mutex>

namespace
{
    constexpr size_t kInferResultLogMaxBytes = 2048;
    std::mutex g_infer_log_mutex;

    std::string TruncateInferResult(const std::string &result_json)
    {
        if (result_json.size() <= kInferResultLogMaxBytes)
        {
            return result_json;
        }
        return result_json.substr(0, kInferResultLogMaxBytes) + "...<truncated>";
    }

    void AppendInferInputFields(std::ostream &out, const hw_buffer_desc_t &desc)
    {
        out << " input_width=" << desc.width
            << " input_height=" << desc.height
            << " input_size=" << desc.size
            << " dma_fd=" << desc.dma_fd
            << " data=" << desc.data
            << " stride=" << desc.stride;
    }
}

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
            {
                std::lock_guard<std::mutex> lock(g_infer_log_mutex);
                std::cerr << "[AlgoInstance] infer skipped"
                          << " algo=" << algo_name_
                          << " version=" << version_
                          << " reason=not_initialized"
                          << " algo_handle=" << algo_handle_
                          << " so_handle=" << so_handle_.get()
                          << std::endl;
                return false;
            }

            state_.store(AlgoInstanceState::Running);

            hw_buffer_desc_t fb_desc{};
            fb_desc.dma_fd = input_desc.dma_fd;
            fb_desc.size = input_desc.size;
            fb_desc.width = input_desc.width;
            fb_desc.height = input_desc.height;
            fb_desc.pixel_format = input_desc.pixel_format;
            fb_desc.dma_buf_fd = input_desc.dma_buf_fd;
            fb_desc.phys_addr = input_desc.phys_addr;
            fb_desc.data = input_desc.data;
            fb_desc.stride = input_desc.stride;

            infer_result_t result{};
            int ret = so_handle_->Infer(
                algo_handle_,
                &fb_desc,
                context_json.empty() ? nullptr : context_json.c_str(),
                &result);

            if (ret == 0 && result.result_json)
            {
                result_json = result.result_json;
                infer_time_us = result.infer_time_us;
                std::lock_guard<std::mutex> lock(g_infer_log_mutex);
                std::cout << "[AlgoInstance] infer result"
                          << " algo=" << algo_name_
                          << " version=" << version_
                          << " infer_time_us=" << infer_time_us;
                AppendInferInputFields(std::cout, fb_desc);
                std::cout << " result_size=" << result_json.size()
                          << " result=" << TruncateInferResult(result_json)
                          << std::endl;
                algo_free_result(&result);
            }
            else
            {
                std::lock_guard<std::mutex> lock(g_infer_log_mutex);
                std::cerr << "[AlgoInstance] infer failed"
                          << " algo=" << algo_name_
                          << " version=" << version_
                          << " ret=" << ret;
                AppendInferInputFields(std::cerr, fb_desc);
                std::cerr << std::endl;
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

#include <cstdlib>

extern "C" {
    void algo_free_result(infer_result_t *result) {
        if (result && result->result_json) {
            std::free(result->result_json);
            result->result_json = nullptr;
            result->result_json_len = 0;
        }
    }
}
