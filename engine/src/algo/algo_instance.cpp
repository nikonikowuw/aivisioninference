// AlgoInstance 实现
#include "algo/algo_instance.h"
#include "logger/logger.h"
#include <mutex>

namespace {
hw_buffer_desc_t BuildAbiBufferDesc(const aivision::pipeline::HwBufferDesc &input_desc)
    {
        hw_buffer_desc_t desc{};
        desc.dma_fd = input_desc.dma_fd;
        desc.size = input_desc.size;
        desc.width = input_desc.width;
        desc.height = input_desc.height;
        desc.pixel_format = input_desc.pixel_format;
        desc.dma_buf_fd = input_desc.dma_buf_fd;
        desc.phys_addr = input_desc.phys_addr;
        desc.data = input_desc.data;
        desc.stride = input_desc.stride;
        desc.buffer_owner = HW_BUFFER_OWNER_ENGINE;
        desc.buffer_type = HW_BUFFER_TYPE_DEFAULT;

        if (input_desc.native_handle == nullptr)
        {
            return desc;
        }

        switch (input_desc.memory_type)
        {
        case aivision::pipeline::HwBufferMemoryType::CVPixelBuffer:
            desc.buffer_type = HW_BUFFER_TYPE_APPLE_NATIVE;
            desc.plat.apple.abi_version = HW_BUFFER_APPLE_ABI_VERSION;
            desc.plat.apple.struct_size = sizeof(hw_buffer_apple_t);
            desc.plat.apple.buffer_kind = HW_BUFFER_APPLE_CVPIXELBUFFER;
            desc.plat.apple.pixel_format = input_desc.pixel_format;
            desc.plat.apple.native_handle = reinterpret_cast<uint64_t>(input_desc.native_handle);
            desc.plat.apple.plane_count = HW_BUFFER_PLANE_COUNT_UNKNOWN;
            break;
        default:
            break;
        }

        return desc;
}
} // namespace

namespace aivision
{
    namespace algo
    {

        AlgoInstance::AlgoInstance(std::shared_ptr<SoHandle> so_handle,
                                   const std::string &algo_name,
                                   const std::string &version)
            : so_handle_(std::move(so_handle)), algo_name_(algo_name), version_(version)
        {
            UpdateAccessTime();
        }

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
            if (!so_handle_)
            {
                LOG_ERROR("[AlgoInstance] infer skipped algo={} version={} reason=no_so_handle",
                         algo_name_, version_);
                return false;
            }

            active_infer_count_.fetch_add(1);
            state_.store(AlgoInstanceState::Running);

            const hw_buffer_desc_t fb_desc = BuildAbiBufferDesc(input_desc);

            infer_result_t result{};
            int ret = 0;
            {
                std::lock_guard<std::mutex> instance_infer_lock(infer_mutex_);
                if (!algo_handle_)
                {
                    if (active_infer_count_.fetch_sub(1) == 1) {
                        state_.store(AlgoInstanceState::Ready);
                    }
                    LOG_ERROR("[AlgoInstance] infer skipped reason=not_initialized algo={}", algo_name_);
                    return false;
                }
                ret = so_handle_->Infer(
                    algo_handle_,
                    &fb_desc,
                    context_json.empty() ? nullptr : context_json.c_str(),
                    &result);
            }

            if (ret == 0 && result.result_json)
            {
                result_json = result.result_json;
                infer_time_us = result.infer_time_us;
                algo_free_result(&result);
            }
            else if (ret != 0)
            {
                LOG_ERROR("[AlgoInstance] infer failed algo={} version={} ret={} "
                         "input_width={} input_height={} input_size={} dma_fd={} data={} stride={}",
                         algo_name_, version_, ret,
                         fb_desc.width, fb_desc.height, fb_desc.size,
                         fb_desc.dma_fd, fb_desc.data, fb_desc.stride);
            }

            if (active_infer_count_.fetch_sub(1) == 1) {
                state_.store(AlgoInstanceState::Ready);
            }
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

        bool AlgoInstance::SupportsFaceLibraryUpdate() const
        {
            return so_handle_ && so_handle_->HasFaceLibraryUpdater();
        }

        bool AlgoInstance::UpdateFaceLibrary(const std::string &face_library_json)
        {
            std::lock_guard<std::mutex> lock(infer_mutex_);
            if (!algo_handle_ || !so_handle_)
            {
                LOG_ERROR("[AlgoInstance] face library update skipped algo={} version={} reason=not_initialized",
                         algo_name_, version_);
                return false;
            }

            if (!so_handle_->HasFaceLibraryUpdater())
            {
                LOG_ERROR("[AlgoInstance] face library update skipped algo={} version={} reason=symbol_not_found",
                         algo_name_, version_);
                return false;
            }

            const int ret = so_handle_->UpdateFaceLibrary(algo_handle_, face_library_json.c_str());
            if (ret != 0)
            {
                LOG_ERROR("[AlgoInstance] face library update failed algo={} version={} ret={} payload_size={}",
                         algo_name_, version_, ret, face_library_json.size());
                return false;
            }

            LOG_INFO("[AlgoInstance] face library updated algo={} version={} payload_size={}",
                     algo_name_, version_, face_library_json.size());
            return true;
        }

        void AlgoInstance::DestroyInternal()
        {
            std::lock_guard<std::mutex> lock(infer_mutex_);
            if (so_handle_ && algo_handle_)
            {
                so_handle_->Destroy(algo_handle_);
                algo_handle_ = nullptr;
            }
            state_.store(AlgoInstanceState::Unloading);
        }

        void AlgoInstance::UpdateAccessTime()
        {
            auto now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                              std::chrono::steady_clock::now().time_since_epoch())
                              .count();
            last_access_timestamp_ms_.store(now_ms);
        }

        int64_t AlgoInstance::GetIdleTimeMs() const
        {
            auto now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                              std::chrono::steady_clock::now().time_since_epoch())
                              .count();
            return now_ms - last_access_timestamp_ms_.load();
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
