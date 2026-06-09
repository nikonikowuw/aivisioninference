#include "pipeline/hal.h"

#include <dlfcn.h>
#include <iostream>
#include <utility>

namespace aivision
{
    namespace pipeline
    {

        bool HALManager::LoadPipeline(const std::string &so_path,
                                      const std::string &config_json)
        {
            Unload();

            if (so_path.empty())
            {
                std::cerr << "HAL shared object path is empty" << std::endl;
                return false;
            }

            dl_handle_ = dlopen(so_path.c_str(), RTLD_NOW | RTLD_LOCAL);
            if (!dl_handle_)
            {
                std::cerr << "Failed to load HAL shared object: " << so_path
                          << ", error=" << dlerror() << std::endl;
                return false;
            }

            auto create_fn = reinterpret_cast<CreatePipelineFunc>(dlsym(dl_handle_, "CreatePipeline"));
            auto destroy_fn = reinterpret_cast<DestroyPipelineFunc>(dlsym(dl_handle_, "DestroyPipeline"));
            if (!create_fn || !destroy_fn)
            {
                std::cerr << "HAL shared object missing CreatePipeline/DestroyPipeline symbols: "
                          << so_path << std::endl;
                dlclose(dl_handle_);
                dl_handle_ = nullptr;
                return false;
            }

            IMediaPipeline *raw = create_fn();
            if (!raw)
            {
                std::cerr << "HAL CreatePipeline returned null: " << so_path << std::endl;
                dlclose(dl_handle_);
                dl_handle_ = nullptr;
                return false;
            }

            pipeline_ = std::unique_ptr<IMediaPipeline, DestroyPipelineFunc>(raw, destroy_fn);
            if (!pipeline_->Initialize(config_json))
            {
                std::cerr << "HAL pipeline initialize failed: " << so_path << std::endl;
                pipeline_.reset();
                dlclose(dl_handle_);
                dl_handle_ = nullptr;
                return false;
            }

            so_path_ = so_path;
            return true;
        }

        void HALManager::Unload()
        {
            if (pipeline_)
            {
                pipeline_->Stop();
                pipeline_.reset();
            }
            if (dl_handle_)
            {
                dlclose(dl_handle_);
                dl_handle_ = nullptr;
            }
            so_path_.clear();
        }

        std::string HALManager::GetLoadedPlatform() const
        {
            if (!pipeline_)
                return "";
            return pipeline_->GetPipelineType();
        }

    } // namespace pipeline
} // namespace aivision
