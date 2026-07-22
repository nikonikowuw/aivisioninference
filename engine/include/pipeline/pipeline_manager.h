#ifndef AIVISION_PIPELINE_PIPELINE_MANAGER_H
#define AIVISION_PIPELINE_PIPELINE_MANAGER_H

#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <vector>

#include <sys/types.h>

#include "pipeline.h"
#include "hal.h"
#include "ring_queue.h"
#include "ffmpeg_fallback_decoder.h"
#include "rtsp_push_stage.h"

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
            bool inference_enabled = false;
            bool playback_enabled = false;
            bool uses_decoder = false;
            bool uses_encoder = false;
            bool egress_observable = true;
            uint64_t total_egress_bytes = 0;
        };

        /// Pipeline 管理器配置
        struct PipelineManagerConfig
        {
            std::string rtsp_push_server = "rtsp://localhost:10554";
            std::string zlm_secret = "";
            bool enable_ffmpeg_fallback = true;
        };

        /// Pipeline 管理器 — 负责所有流的生命周期管理
        class PipelineManager
        {
        public:
            explicit PipelineManager(PipelineManagerConfig config = {});
            ~PipelineManager();

            /// 设置推理 Worker 使用的流队列管理器
            void SetStreamQueueManager(StreamQueueManager *queue_mgr) { queue_mgr_ = queue_mgr; }

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
            struct MediaRuntimeState
            {
                bool inference_enabled = false;
                bool playback_enabled = false;
                bool uses_decoder = false;
                bool uses_encoder = false;
                bool egress_observable = true;
                std::shared_ptr<RtspPushStage> pusher;
            };

            /// 创建并配置 Stage 的辅助方法 (占位，后续任务实现具体 Stage)
            std::unique_ptr<Stage> CreateInferenceStage(const std::string &device_id);
            std::unique_ptr<Stage> CreatePlaybackStage(const std::string &device_id);

            std::string BuildPushURL(const std::string &device_id) const;
            bool StartFFmpegFallback(const std::string &device_id, const std::string &rtsp_url);
            bool StartFFmpegInferenceFallback(const std::string &device_id, const std::string &rtsp_url, RingQueue *infer_queue);
            void StopFFmpegFallback(const std::string &device_id);

            PipelineManagerConfig config_;
            mutable std::mutex mutex_;
            std::map<std::string, std::unique_ptr<Pipeline>> pipelines_;
            StreamQueueManager *queue_mgr_{nullptr};

            // HAL 管理器，供所有 Pipeline 共享或每个 Pipeline 独立？
            // 设计上 IMediaPipeline 是一路流一个实例。
            std::map<std::string, std::unique_ptr<HALManager>> hal_managers_;
            std::map<std::string, pid_t> ffmpeg_fallbacks_;
            std::map<std::string, std::unique_ptr<FFmpegFallbackDecoder>> ffmpeg_infer_fallbacks_;
            std::map<std::string, MediaRuntimeState> media_runtime_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_PIPELINE_MANAGER_H
