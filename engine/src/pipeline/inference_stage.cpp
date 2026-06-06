#include "pipeline/inference_stage.h"

namespace aivision {
namespace pipeline {

InferenceStage::InferenceStage(WorkerPool* pool) : pool_(pool) {}

bool InferenceStage::Init(const StageContext& ctx) {
    ctx_ = ctx;
    return pool_ != nullptr;
}

bool InferenceStage::Run() {
    running_.store(true);
    // 注意：InferenceStage 本身不启动线程，而是依赖 WorkerPool 轮询 Pipeline 的 RingQueue
    return true;
}

void InferenceStage::Stop() {
    running_.store(false);
}

void InferenceStage::PushFrame(const FrameContext& frame) {
    // 推理帧由 WorkerPool 通过 ctx_.queue->TryPop 获取
}

}  // namespace pipeline
}  // namespace aivision
