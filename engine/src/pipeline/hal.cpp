#include "pipeline/hal.h"
#include "logger/logger.h"

#if defined(__APPLE__)
#include "hal/macos/videotoolbox_pipeline.h"
#elif defined(AIVISION_WITH_RKMPP)
#include "hal/rockchip/rkmpp_pipeline.h"
#endif

namespace aivision
{
    namespace pipeline
    {

        bool HALManager::LoadPipeline(const std::string &config_json)
        {
            Unload();

#if defined(__APPLE__)
            pipeline_ = std::make_unique<VideoToolboxPipeline>();
#elif defined(AIVISION_WITH_RKMPP)
            pipeline_ = std::make_unique<hal::rockchip::RKMPPPipeline>();
#else
            LOG_ERROR("No platform HAL pipeline available at compile time");
            return false;
#endif

            if (!pipeline_->Initialize(config_json))
            {
                LOG_ERROR("HAL pipeline Initialize failed");
                pipeline_.reset();
                return false;
            }

            return true;
        }

        void HALManager::Unload()
        {
            if (pipeline_)
            {
                pipeline_->Stop();
                pipeline_.reset();
            }
        }

        std::string HALManager::GetLoadedPlatform() const
        {
            if (!pipeline_)
                return "";
            return pipeline_->GetPipelineType();
        }

    } // namespace pipeline
} // namespace aivision
