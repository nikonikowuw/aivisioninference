#ifndef AIVISION_ALGO_ABI_CONTRACT_H
#define AIVISION_ALGO_ABI_CONTRACT_H

// ABI 契约规范 — 所有算法 .so 必须遵守的 C ABI。
//
// 必需符号 (必须导出)：
//   detector_init      — 初始化算法上下文
//   detector_infer     — 对一帧执行推理
//   detector_destroy   — 销毁算法上下文
//
// 可选符号 (可导出用于元信息)：
//   detector_version   — 算法版本号
//   detector_name      — 算法名称
//   detector_self_test — 自检函数
//   detector_update_face_library — 热更新算法实例内存人脸库快照
//
// 版本: 2.0.0 — v2 新增 buffer_type 区分 + 平台原生句柄载体，
// 为 macOS CVPixelBufferRef / IOSurfaceRef 提供跨 ABI 传递能力。
// 布局与 aivision-algorithms/engine/include/algo/abi_contract.h 保持严格一致。

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C"
{
#endif

    // ============================================================
    // v2: 缓冲区类型枚举
    // ============================================================

    typedef enum
    {
        HW_BUFFER_TYPE_DEFAULT = 0,
        HW_BUFFER_TYPE_ASCEND_DVPP = 1,
        HW_BUFFER_TYPE_ASCEND_DEVICE = 2,
        HW_BUFFER_TYPE_ROCKCHIP_MPP = 3,
        HW_BUFFER_TYPE_ROCKCHIP_RGA = 4,
        HW_BUFFER_TYPE_CUDA = 5,
        HW_BUFFER_TYPE_APPLE_NATIVE = 6,
    } hw_buffer_type_t;

    typedef enum
    {
        HW_BUFFER_OWNER_ENGINE = 0,
        HW_BUFFER_OWNER_ALGORITHM = 1,
    } hw_buffer_owner_t;

    typedef enum
    {
        HW_BUFFER_APPLE_NONE = 0,
        HW_BUFFER_APPLE_CVPIXELBUFFER = 1,
        HW_BUFFER_APPLE_IOSURFACE = 2,
    } hw_buffer_apple_kind_t;

#define HW_BUFFER_APPLE_ABI_VERSION 1
#define HW_BUFFER_PLANE_COUNT_UNKNOWN 0

    // ============================================================
    // 平台原生缓冲区载荷
    // ============================================================

    typedef struct
    {
        int32_t device_id;
        int32_t memory_subtype;
        uint32_t pixel_format;
        uint32_t aligned_width;
        uint32_t aligned_height;
        uint32_t alignment;
        int32_t cache_synced;
        int32_t color_space;
        uint32_t reserved_flags;
        void *data_ptr;
        uint32_t reserved_pad[3];
    } hw_buffer_ascend_t;

    typedef struct
    {
        int32_t buffer_subtype;
        int32_t hor_stride;
        int32_t ver_stride;
        uint32_t reserved_pad[13];
    } hw_buffer_rockchip_t;

    typedef struct
    {
        uint32_t reserved_pad[16];
    } hw_buffer_cuda_t;

    // Apple CoreVideo / IOSurface 原生缓冲区载荷。
    // native_handle 持有 Engine 借出的 retained CVPixelBufferRef (或 IOSurfaceRef)，
    // 算法在同步 detector_infer 调用期间 borrow，不可 release 或存储到调用之外。
    typedef struct
    {
        uint16_t abi_version;   // HW_BUFFER_APPLE_ABI_VERSION
        uint16_t struct_size;   // sizeof(hw_buffer_apple_t)
        uint32_t buffer_kind;   // hw_buffer_apple_kind_t
        uint32_t pixel_format;  // CoreVideo / IOSurface FourCC (如 kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange)
        uint32_t plane_count;   // 1 (BGRA)、2 (NV12)，0=由算法查询原生对象
        uint64_t native_handle; // retained CVPixelBufferRef / IOSurfaceRef
        uint32_t plane_stride[2];
        uint32_t plane_offset[2];
        uint64_t synchronization; // 保留同步 token
        uint32_t reserved_pad[3]; // 填充至 64 字节
    } hw_buffer_apple_t;

    // ============================================================
    // 类型定义
    // ============================================================

    /// 算法上下文句柄 (不透明指针)
    typedef struct algo_context_t *algo_handle_t;

    /// 硬件缓冲区描述符 v2 — 与 HwBufferDesc 对应，
    /// 新增 buffer_type / plat 支持平台原生句柄零拷贝传递。
    /// 布局必须与算法侧 aivision-algorithms/engine/include/algo/abi_contract.h 严格一致。
    typedef struct
    {
        // v1 字段 (ABI 前缀兼容)
        int dma_fd;
        size_t size;
        uint32_t width;
        uint32_t height;
        uint32_t pixel_format;
        int dma_buf_fd;
        uint64_t phys_addr;
        void *data;
        uint32_t stride;

        // v2 字段
        int32_t buffer_type;   // hw_buffer_type_t
        int32_t buffer_owner;  // HW_BUFFER_OWNER_ENGINE / HW_BUFFER_OWNER_ALGORITHM
        uint32_t reserved_flags;

        union
        {
            hw_buffer_ascend_t ascend;
            hw_buffer_rockchip_t rockchip;
            hw_buffer_cuda_t cuda;
            hw_buffer_apple_t apple;
        } plat;

        int64_t reserved_padding[2];
    } hw_buffer_desc_t;

// 编译期结构体尺寸校验必须覆盖 Engine 的 C++ 构建和算法侧 C 构建。
#if defined(__cplusplus)
    static_assert(sizeof(hw_buffer_ascend_t) == 64, "hw_buffer_ascend_t must be 64 bytes");
    static_assert(sizeof(hw_buffer_rockchip_t) == 64, "hw_buffer_rockchip_t must be 64 bytes");
    static_assert(sizeof(hw_buffer_cuda_t) == 64, "hw_buffer_cuda_t must be 64 bytes");
    static_assert(sizeof(hw_buffer_apple_t) == 64, "hw_buffer_apple_t must be 64 bytes");
    static_assert(sizeof(hw_buffer_desc_t) == 144, "hw_buffer_desc_t must be 144 bytes");
#elif defined(__STDC_VERSION__) && __STDC_VERSION__ >= 201112L
    _Static_assert(sizeof(hw_buffer_ascend_t) == 64, "hw_buffer_ascend_t must be 64 bytes");
    _Static_assert(sizeof(hw_buffer_rockchip_t) == 64, "hw_buffer_rockchip_t must be 64 bytes");
    _Static_assert(sizeof(hw_buffer_cuda_t) == 64, "hw_buffer_cuda_t must be 64 bytes");
    _Static_assert(sizeof(hw_buffer_apple_t) == 64, "hw_buffer_apple_t must be 64 bytes");
    _Static_assert(sizeof(hw_buffer_desc_t) == 144, "hw_buffer_desc_t must be 144 bytes");
#endif

    /// 推理结果
    typedef struct
    {
        /// 结果 JSON 字符串 (由算法内部 malloc，由调用者 free)
        char *result_json;

        /// 结果 JSON 长度
        size_t result_json_len;

        /// 推理耗时 (微秒)
        uint32_t infer_time_us;

        /// 预留字段
        int reserved[4];
    } infer_result_t;

    // ============================================================
    // 必需符号 (Must Export)
    // ============================================================

    /// 初始化算法上下文。
    /// @param config_json 算法配置参数的 JSON 字符串 (由调用者管理生命周期)
    /// @return 算法句柄，失败返回 NULL
    algo_handle_t detector_init(const char *config_json);

    /// 对一帧执行推理。
    /// @param handle  算法句柄 (由 detector_init 返回)
    /// @param input   输入帧的硬件缓冲区描述符 (零拷贝)
    /// @param context_json  算法串联上下文 JSON (上一级算法输出，可为 NULL)
    /// @param result  输出推理结果
    /// @return 0 成功，非 0 失败
    int detector_infer(algo_handle_t handle,
                       const hw_buffer_desc_t *input,
                       const char *context_json,
                       infer_result_t *result);

    /// 销毁算法上下文，释放所有资源。
    /// @param handle 算法句柄
    void detector_destroy(algo_handle_t handle);

    // ============================================================
    // 可选符号 (Optional)
    // ============================================================

    /// 获取算法版本号。
    /// @return 版本字符串 (如 "1.2.0")，由算法内部管理生命周期
    const char *detector_version(void);

    /// 获取算法名称。
    /// @return 名称字符串 (如 "person_detector")，由算法内部管理生命周期
    const char *detector_name(void);

    /// 执行自检。
    /// @return 0 自检通过，非 0 自检失败
    int detector_self_test(void);

    /// 热更新算法实例内存人脸库快照。
    /// 该接口为可选符号。Engine 只转发完整快照 JSON，不解释新增/删除等业务语义。
    /// @param handle 算法句柄
    /// @param face_library_json 完整人脸库快照 JSON，由调用者管理生命周期
    /// @return 0 更新成功，非 0 更新失败或算法不支持
    int detector_update_face_library(algo_handle_t handle, const char *face_library_json);

    // ============================================================
    // 辅助函数 (由引擎提供，算法可调用)
    // ============================================================

    /// 释放推理结果中的 JSON 字符串。
    /// 当调用者使用完 result_json 后，调用此函数释放内存。
    void algo_free_result(infer_result_t *result);

#ifdef __cplusplus
}
#endif

#endif // AIVISION_ALGO_ABI_CONTRACT_H
