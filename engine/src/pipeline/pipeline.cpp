#include "pipeline/pipeline.h"
#include "logger/logger.h"
#include <algorithm>

namespace aivision
{
    namespace pipeline
    {

        Pipeline::Pipeline(const std::string &device_id, size_t queue_capacity)
            : device_id_(device_id), queue_(queue_capacity)
        {
            ctx_.device_id = device_id;
            ctx_.queue = &queue_;
        }

        Pipeline::~Pipeline()
        {
            Stop();
        }

        bool Pipeline::AddStage(std::shared_ptr<Stage> stage)
        {
            if (!stage)
                return false;

            std::lock_guard<std::mutex> lock(mutex_);

            // 检查是否已存在同名 Stage
            auto it = std::find_if(stages_.begin(), stages_.end(),
                                   [&stage](const auto &s)
                                   { return s->GetName() == stage->GetName(); });

            if (it != stages_.end())
            {
                LOG_ERROR("Stage {} already exists in pipeline {}", stage->GetName(), device_id_);
                return false;
            }

            if (!stage->Init(ctx_))
            {
                LOG_ERROR("Failed to initialize stage {} in pipeline {}", stage->GetName(), device_id_);
                return false;
            }

            if (running_.load())
            {
                if (!stage->Run())
                {
                    LOG_ERROR("Failed to start stage {} in pipeline {}", stage->GetName(), device_id_);
                    return false;
                }
            }

            stages_.push_back(std::move(stage));
            return true;
        }

        bool Pipeline::AddStage(std::unique_ptr<Stage> stage)
        {
            return AddStage(std::shared_ptr<Stage>(std::move(stage)));
        }

        bool Pipeline::RemoveStage(const std::string &name)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it = std::find_if(stages_.begin(), stages_.end(),
                                   [&name](const auto &s)
                                   { return s->GetName() == name; });

            if (it == stages_.end())
            {
                return false;
            }

            (*it)->Stop();
            stages_.erase(it);
            return true;
        }

        bool Pipeline::Start()
        {
            std::lock_guard<std::mutex> lock(mutex_);
            if (running_.load())
                return true;

            size_t started = 0;
            for (auto &stage : stages_)
            {
                if (!stage->Run())
                {
                    LOG_ERROR("Failed to run stage {} during pipeline start, rolling back", stage->GetName());
                    // 回滚：停止已启动的 Stage
                    for (size_t i = 0; i < started; ++i)
                    {
                        stages_[i]->Stop();
                    }
                    return false;
                }
                ++started;
            }

            running_.store(true);
            return true;
        }

        void Pipeline::Stop()
        {
            std::lock_guard<std::mutex> lock(mutex_);
            if (!running_.load() && stages_.empty())
                return;

            for (auto &stage : stages_)
            {
                stage->Stop();
            }

            stages_.clear();
            queue_.Clear();
            running_.store(false);
        }

        std::vector<std::string> Pipeline::GetStageNames() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            std::vector<std::string> names;
            for (const auto &stage : stages_)
            {
                names.push_back(stage->GetName());
            }
            return names;
        }

    } // namespace pipeline
} // namespace aivision
