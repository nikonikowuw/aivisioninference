#include "pipeline/pipeline_manager.h"
#include "pipeline/inference_stage.h"
#include "pipeline/encoder_stage.h"
#include "pipeline/rtsp_push_stage.h"
#include <chrono>
#include <csignal>
#include <cerrno>
#include <atomic>
#include <iostream>
#include <memory>
#include <sys/wait.h>
#include <thread>
#include <utility>
#include <unistd.h>

namespace aivision
{
    namespace pipeline
    {
        namespace
        {
            void TerminateChildProcess(pid_t pid)
            {
                if (pid <= 0)
                    return;

                kill(pid, SIGTERM);
                for (int i = 0; i < 20; ++i)
                {
                    pid_t result = waitpid(pid, nullptr, WNOHANG);
                    if (result == pid || (result < 0 && errno == ECHILD))
                        return;
                    std::this_thread::sleep_for(std::chrono::milliseconds(100));
                }

                kill(pid, SIGKILL);
                waitpid(pid, nullptr, 0);
            }

            FrameContext MakeFrameContext(const std::string &device_id, HwBufferPtr frame, uint64_t frame_seq)
            {
                FrameContext ctx;
                ctx.buffer = std::move(frame);
                ctx.timestamp_ns = static_cast<uint64_t>(
                    std::chrono::duration_cast<std::chrono::nanoseconds>(
                        std::chrono::system_clock::now().time_since_epoch())
                        .count());
                ctx.task_id = device_id;
                ctx.frame_seq = frame_seq;
                return ctx;
            }
        }

        PipelineManager::PipelineManager(PipelineManagerConfig config)
            : config_(std::move(config)) {}

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
            auto create_ffmpeg_fallback = [&]() {
                if (!config_.enable_ffmpeg_fallback)
                    return false;

                if (enable_infer)
                {
                    if (!queue_mgr_)
                        return false;
                    RingQueue *infer_queue = queue_mgr_->CreateStream(device_id);
                    if (!StartFFmpegInferenceFallback(device_id, rtsp_url, infer_queue))
                    {
                        queue_mgr_->RemoveStream(device_id);
                        return false;
                    }
                    pipelines_[device_id] = std::move(pipeline);
                    std::cout << "FFmpeg software inference fallback pipeline created for device: "
                              << device_id << std::endl;
                    return true;
                }

                if (!StartFFmpegFallback(device_id, rtsp_url))
                    return false;

                pipelines_[device_id] = std::move(pipeline);
                std::cout << "FFmpeg relay fallback pipeline created for device: " << device_id << std::endl;
                return true;
            };

            // 2. 创建并启动 HAL (拉流/硬件解码)。推理场景下 fallback 也必须是真实 HAL。
            std::vector<std::pair<std::string, std::string>> hal_candidates;
            if (!config_.hal_so_path.empty())
            {
                hal_candidates.emplace_back("primary", config_.hal_so_path);
            }
            if (!config_.fallback_hal_so_path.empty() && config_.fallback_hal_so_path != config_.hal_so_path)
            {
                hal_candidates.emplace_back("fallback", config_.fallback_hal_so_path);
            }
            if (hal_candidates.empty())
            {
                std::cerr << "HAL .so path is not configured for device " << device_id << std::endl;
                return create_ffmpeg_fallback();
            }

            std::unique_ptr<HALManager> hal;
            IMediaPipeline *media_pipeline = nullptr;
            for (const auto &[kind, hal_path] : hal_candidates)
            {
                auto candidate = std::make_unique<HALManager>();
                if (!candidate->LoadPipeline(hal_path, config_.hal_config_json))
                {
                    std::cerr << "Failed to load " << kind << " HAL pipeline for device "
                              << device_id << ", hal_so=" << hal_path << std::endl;
                    continue;
                }
                media_pipeline = candidate->GetPipeline();
                if (!media_pipeline)
                {
                    std::cerr << kind << " HAL pipeline is null for device " << device_id << std::endl;
                    continue;
                }
                hal = std::move(candidate);
                std::cout << "Loaded " << kind << " HAL for device " << device_id
                          << ", hal_so=" << hal_path << std::endl;
                break;
            }
            if (!hal || !media_pipeline)
            {
                std::cerr << "No usable HAL pipeline for device " << device_id << std::endl;
                return create_ffmpeg_fallback();
            }
            RingQueue *infer_queue = nullptr;
            if (enable_infer && queue_mgr_)
            {
                infer_queue = queue_mgr_->CreateStream(device_id);
                std::cout << "[PipelineManager] infer queue created device=" << device_id << std::endl;
            }
            auto frame_counter = std::make_shared<std::atomic<uint64_t>>(0);
            media_pipeline->SetFrameCallback([queue = pipeline->GetQueue(), infer_queue, device_id, frame_counter](HwBufferPtr frame) {
                if (!frame)
                    return;
                uint64_t count = frame_counter->fetch_add(1) + 1;
                FrameContext ctx = MakeFrameContext(device_id, frame, count);
                if (count == 1 || count % 100 == 0)
                {
                    std::cout << "[PipelineManager] frame received"
                              << " device=" << device_id
                              << " count=" << count
                              << " infer_queue=" << (infer_queue ? "yes" : "no")
                              << std::endl;
                }
                if (queue)
                {
                    FrameContext pipeline_ctx = ctx;
                    queue->Push(std::move(pipeline_ctx));
                }
                if (infer_queue)
                {
                    infer_queue->Push(std::move(ctx));
                }
            });
            if (!media_pipeline->Start(rtsp_url))
            {
                std::cerr << "Failed to start HAL stream for device " << device_id
                          << ", url=" << rtsp_url << std::endl;
                if (infer_queue && queue_mgr_)
                    queue_mgr_->RemoveStream(device_id);
                return create_ffmpeg_fallback();
            }

            // 3. 根据参数添加初始 Stage
            if (enable_infer)
            {
                // pipeline->AddStage(CreateInferenceStage(device_id));
            }

            if (enable_playback)
            {
                std::string push_url = BuildPushURL(device_id);

                auto encoder = std::make_shared<EncoderStage>(media_pipeline);
                auto pusher = std::make_shared<RtspPushStage>(push_url, encoder);
                if (!pipeline->AddStage(encoder) || !pipeline->AddStage(pusher))
                {
                    media_pipeline->Stop();
                    if (infer_queue && queue_mgr_)
                        queue_mgr_->RemoveStream(device_id);
                    return false;
                }
            }

            // 4. 启动 Pipeline
            if (!pipeline->Start())
            {
                media_pipeline->Stop();
                if (infer_queue && queue_mgr_)
                    queue_mgr_->RemoveStream(device_id);
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
            StopFFmpegFallback(device_id);
            auto hal_it = hal_managers_.find(device_id);
            if (hal_it != hal_managers_.end() && hal_it->second->GetPipeline())
            {
                hal_it->second->GetPipeline()->Stop();
            }
            pipelines_.erase(it);
            hal_managers_.erase(device_id);
            if (queue_mgr_)
                queue_mgr_->RemoveStream(device_id);

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
            if (it_hal == hal_managers_.end())
                return false;

            std::string push_url = BuildPushURL(device_id);
            auto encoder = std::make_shared<EncoderStage>(it_hal->second->GetPipeline());
            auto pusher = std::make_shared<RtspPushStage>(push_url, encoder);

            return p->AddStage(encoder) && p->AddStage(pusher);
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
            for (auto &[device_id, p] : pipelines_)
            {
                (void)device_id;
                p->Stop();
            }
            for (auto &[device_id, hal] : hal_managers_)
            {
                (void)device_id;
                if (hal && hal->GetPipeline())
                    hal->GetPipeline()->Stop();
            }
            for (auto &[device_id, pid] : ffmpeg_fallbacks_)
            {
                (void)device_id;
                TerminateChildProcess(pid);
            }
            for (auto &[device_id, decoder] : ffmpeg_infer_fallbacks_)
            {
                (void)device_id;
                if (decoder)
                    decoder->Stop();
            }
            pipelines_.clear();
            hal_managers_.clear();
            ffmpeg_fallbacks_.clear();
            ffmpeg_infer_fallbacks_.clear();
        }

        std::string PipelineManager::BuildPushURL(const std::string &device_id) const
        {
            std::string push_url = config_.rtsp_push_server;
            if (push_url.empty())
                push_url = "rtsp://localhost:10554";
            if (!push_url.empty() && push_url.back() == '/')
                push_url.pop_back();
            return push_url + "/live/" + device_id;
        }

        bool PipelineManager::StartFFmpegFallback(const std::string &device_id, const std::string &rtsp_url)
        {
            std::string push_url = BuildPushURL(device_id);
            pid_t pid = fork();
            if (pid < 0)
            {
                std::cerr << "Failed to fork FFmpeg fallback for device " << device_id << std::endl;
                return false;
            }

            if (pid == 0)
            {
                execlp("ffmpeg", "ffmpeg",
                       "-hide_banner", "-loglevel", "warning",
                       "-analyzeduration", "5000000",
                       "-probesize", "10000000",
                       "-rtsp_transport", "tcp",
                       "-i", rtsp_url.c_str(),
                       "-an", "-c:v", "copy",
                       "-f", "rtsp", "-rtsp_transport", "tcp",
                       push_url.c_str(),
                       static_cast<char *>(nullptr));
                _exit(127);
            }

            std::this_thread::sleep_for(std::chrono::milliseconds(500));
            int status = 0;
            pid_t exited = waitpid(pid, &status, WNOHANG);
            if (exited == pid)
            {
                std::cerr << "FFmpeg fallback exited immediately for device " << device_id
                          << ", status=" << status << std::endl;
                return false;
            }

            ffmpeg_fallbacks_[device_id] = pid;
            std::cout << "FFmpeg fallback started for device: " << device_id
                      << ", pid=" << pid
                      << ", push_url=" << push_url << std::endl;
            return true;
        }

        bool PipelineManager::StartFFmpegInferenceFallback(const std::string &device_id, const std::string &rtsp_url, RingQueue *infer_queue)
        {
            if (!infer_queue)
                return false;

            auto decoder = std::make_unique<FFmpegFallbackDecoder>();
            auto frame_counter = std::make_shared<std::atomic<uint64_t>>(0);
            bool started = decoder->Start(rtsp_url, device_id, 640, 360,
                                          [infer_queue, device_id, frame_counter](HwBufferPtr frame) {
                                              if (!frame)
                                                  return;
                                              uint64_t count = frame_counter->fetch_add(1) + 1;
                                              infer_queue->Push(MakeFrameContext(device_id, std::move(frame), count));
                                              if (count == 1 || count % 100 == 0)
                                              {
                                                  std::cout << "[PipelineManager] ffmpeg fallback frame queued"
                                                            << " device=" << device_id
                                                            << " count=" << count << std::endl;
                                              }
                                          });
            if (!started)
                return false;

            ffmpeg_infer_fallbacks_[device_id] = std::move(decoder);
            return true;
        }

        void PipelineManager::StopFFmpegFallback(const std::string &device_id)
        {
            auto it = ffmpeg_fallbacks_.find(device_id);
            if (it != ffmpeg_fallbacks_.end())
            {
                pid_t pid = it->second;
                TerminateChildProcess(pid);
                ffmpeg_fallbacks_.erase(it);
            }

            auto infer_it = ffmpeg_infer_fallbacks_.find(device_id);
            if (infer_it != ffmpeg_infer_fallbacks_.end())
            {
                infer_it->second->Stop();
                ffmpeg_infer_fallbacks_.erase(infer_it);
            }
        }

        std::unique_ptr<Stage> PipelineManager::CreateInferenceStage(const std::string &device_id)
        {
            (void)device_id;
            // 占位，任务 7.x 中实现
            return nullptr;
        }

        std::unique_ptr<Stage> PipelineManager::CreatePlaybackStage(const std::string &device_id)
        {
            (void)device_id;
            // 占位，任务 8/9 中实现
            return nullptr;
        }

    } // namespace pipeline
} // namespace aivision
