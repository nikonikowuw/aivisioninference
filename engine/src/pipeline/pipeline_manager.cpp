#include "pipeline/pipeline_manager.h"
#include "pipeline/inference_stage.h"
#include "pipeline/encoder_stage.h"
#include "pipeline/rtsp_push_stage.h"
#include <iostream>

namespace aivision
{
    namespace pipeline
    {

        PipelineManager::PipelineManager() {}

        PipelineManager::~PipelineManager()
        {
            StopAll();
        }

        bool PipelineManager::CreatePipeline(const std::string &device_id,
                                             const std::string &rtsp_url,
                                             bool enable_infer,
                                             bool enable_playback)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            if (pipelines_.find(device_id) != pipelines_.end())
            {
                std::cerr << "Pipeline for device " << device_id << " already exists" << std::endl;
                return false;
            }

            // 1. 创建 Pipeline 实例
            auto pipeline = std::make_unique<Pipeline>(device_id);

            // 2. 创建并启动 HAL (拉流/解码)
            // 注意：这里需要配合 HALManager。目前简单处理。
            auto hal = std::make_unique<HALManager>();
            // TODO: 从配置中获取 hal_so_path
            // if (!hal->LoadPipeline(hal_so_path, hal_config)) { ... }

            // 3. 根据参数添加初始 Stage
            if (enable_infer)
            {
                // pipeline->AddStage(CreateInferenceStage(device_id));
            }

            if (enable_playback)
            {
                // pipeline->AddStage(CreatePlaybackStage(device_id));
            }

            // 4. 启动 Pipeline
            if (!pipeline->Start())
            {
                return false;
            }

            pipelines_[device_id] = std::move(pipeline);
            hal_managers_[device_id] = std::move(hal);

            std::cout << "Pipeline created for device: " << device_id << std::endl;
            return true;
        }

        bool PipelineManager::DestroyPipeline(const std::string &device_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it = pipelines_.find(device_id);
            if (it == pipelines_.end())
            {
                return false;
            }

            it->second->Stop();
            pipelines_.erase(it);
            hal_managers_.erase(device_id);

            std::cout << "Pipeline destroyed for device: " << device_id << std::endl;
            return true;
        }

        bool PipelineManager::EnablePlayback(const std::string &device_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it_pipe = pipelines_.find(device_id);
            if (it_pipe == pipelines_.end())
                return false;
            auto *p = it_pipe->second.get();

            // 获取 HAL
            auto it_hal = hal_managers_.find(device_id);
            if (it_hal == hal_managers_.end()) return false;
            
            std::string push_url = "rtsp://zlm:554/live/" + device_id;
            
            // 1. 创建编码 Stage (shared_ptr 供 RtspPushStage 引用)
            auto encoder = std::make_shared<EncoderStage>(it_hal->second->GetPipeline());
            
            // 2. 创建推流 Stage (shared_ptr 引用 encoder)
            auto pusher = std::make_shared<RtspPushStage>(push_url, encoder);
            
            if (!p->AddStage(encoder)) return false;
            if (!p->AddStage(pusher)) return false;
            
            return true;
        }

        bool PipelineManager::DisablePlayback(const std::string &device_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it = pipelines_.find(device_id);
            if (it == pipelines_.end())
                return false;

            it->second->RemoveStage("RtspPushStage");
            it->second->RemoveStage("EncoderStage");
            return true;
        }

        bool PipelineManager::EnableInfer(const std::string &device_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it = pipelines_.find(device_id);
            if (it == pipelines_.end())
                return false;

            // return it->second->AddStage(CreateInferenceStage(device_id));
            return true;
        }

        bool PipelineManager::DisableInfer(const std::string &device_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it = pipelines_.find(device_id);
            if (it == pipelines_.end())
                return false;

            return it->second->RemoveStage("InferenceStage");
        }

        std::vector<PipelineStatus> PipelineManager::ListPipelines() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            std::vector<PipelineStatus> statuses;

            for (const auto &[id, p] : pipelines_)
            {
                PipelineStatus s;
                s.device_id = id;
                s.is_running = true; // TODO: 细化状态
                s.active_stages = p->GetStageNames();
                s.queue_size = p->GetQueue()->Size();
                s.queue_capacity = p->GetQueue()->Capacity();
                statuses.push_back(s);
            }

            return statuses;
        }

        Pipeline *PipelineManager::GetPipeline(const std::string &device_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = pipelines_.find(device_id);
            if (it == pipelines_.end())
                return nullptr;
            return it->second.get();
        }

        void PipelineManager::StopAll()
        {
            std::lock_guard<std::mutex> lock(mutex_);
            for (auto &[id, p] : pipelines_)
            {
                p->Stop();
            }
            pipelines_.clear();
            hal_managers_.clear();
        }

        std::unique_ptr<Stage> PipelineManager::CreateInferenceStage(const std::string &device_id)
        {
            // 占位，任务 7.x 中实现
            return nullptr;
        }

        std::unique_ptr<Stage> PipelineManager::CreatePlaybackStage(const std::string &device_id)
        {
            // 占位，任务 8/9 中实现
            return nullptr;
        }

    } // namespace pipeline
} // namespace aivision
