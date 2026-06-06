#include "pipeline/pipeline.h"
#include <algorithm>
#include <iostream>

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
                std::cerr << "Stage " << stage->GetName() << " already exists in pipeline "
                          << device_id_ << std::endl;
                return false;
            }

            if (!stage->Init(ctx_))
            {
                std::cerr << "Failed to initialize stage " << stage->GetName()
                          << " in pipeline " << device_id_ << std::endl;
                return false;
            }

            if (running_.load())
            {
                if (!stage->Run())
                {
                    std::cerr << "Failed to start stage " << stage->GetName()
                              << " in pipeline " << device_id_ << std::endl;
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
                    std::cerr << "Failed to run stage " << stage->GetName()
                              << " during pipeline start, rolling back" << std::endl;
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
