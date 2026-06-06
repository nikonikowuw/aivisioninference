#ifndef AIVISION_PIPELINE_PIPELINE_MANAGER_H
#define AIVISION_PIPELINE_PIPELINE_MANAGER_H

#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <vector>

#include "pipeline.h"
#include "hal.h"

namespace aivision
{
    namespace pipeline
    {

        /// Pipeline 状态快照
        struct PipelineStatus
        {
            std::string device_id;
            bool is_running;
            std::vector<std::string> active_stages;
            size_t queue_size;
            size_t queue_capacity;
        };

        /// Pipeline 管理器 — 负责所有流的生命周期管理
        class PipelineManager
        {
        public:
            PipelineManager();
            ~PipelineManager();

            /// 创建并启动 Pipeline
            bool CreatePipeline(const std::string &device_id,
                                const std::string &rtsp_url,
                                bool enable_infer,
                                bool enable_playback);

            /// 销毁 Pipeline
            bool DestroyPipeline(const std::string &device_id);

            /// 启用播放 (编码+推流)
            bool EnablePlayback(const std::string &device_id);

            /// 禁用播放
            bool DisablePlayback(const std::string &device_id);

            /// 启用推理
            bool EnableInfer(const std::string &device_id);

            /// 禁用推理
            bool DisableInfer(const std::string &device_id);

            /// 获取 Pipeline 列表
            std::vector<PipelineStatus> ListPipelines() const;

            /// 获取指定 Pipeline
            Pipeline *GetPipeline(const std::string &device_id);

            /// 停止所有 Pipeline
            void StopAll();

        private:
            /// 创建并配置 Stage 的辅助方法 (占位，后续任务实现具体 Stage)
            std::unique_ptr<Stage> CreateInferenceStage(const std::string &device_id);
            std::unique_ptr<Stage> CreatePlaybackStage(const std::string &device_id);

            mutable std::mutex mutex_;
            std::map<std::string, std::unique_ptr<Pipeline>> pipelines_;

            // HAL 管理器，供所有 Pipeline 共享或每个 Pipeline 独立？
            // 设计上 IMediaPipeline 是一路流一个实例。
            std::map<std::string, std::unique_ptr<HALManager>> hal_managers_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_PIPELINE_MANAGER_H
