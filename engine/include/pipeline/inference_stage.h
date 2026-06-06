#ifndef AIVISION_PIPELINE_INFERENCE_STAGE_H
#define AIVISION_PIPELINE_INFERENCE_STAGE_H

#include "pipeline.h"
#include "worker_pool.h"

namespace aivision {
namespace pipeline {

/// 推理 Stage — 将帧数据桥接到 WorkerPool 进行异步推理
class InferenceStage : public Stage {
public:
    explicit InferenceStage(WorkerPool* pool);
    ~InferenceStage() override = default;

    bool Init(const StageContext& ctx) override;
    bool Run() override;
    void Stop() override;
    void PushFrame(const FrameContext& frame) override;
    
    std::string GetName() const override { return "InferenceStage"; }
    bool IsRunning() const override { return running_.load(); }

private:
    WorkerPool* pool_ = nullptr;
    StageContext ctx_;
    std::atomic<bool> running_{false};
};

}  // namespace pipeline
}  // namespace aivision

#endif  // AIVISION_PIPELINE_INFERENCE_STAGE_H
