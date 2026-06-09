#ifndef AIVISION_ALGO_SO_HANDLE_H
#define AIVISION_ALGO_SO_HANDLE_H

// SoHandle — RAII 封装动态库 (dlopen/dlsym/dlclose)。
// 核心设计：
//   1. 构造时 dlopen 加载 .so 并校验必需符号。
//   2. 析构时自动 dlclose 释放。
//   3. 提供安全的符号查询 (dlsym + 异常捕获)。
//   4. 支持移动语义 (不可拷贝)。

#include <atomic>
#include <stdexcept>
#include <string>
#include <functional>
#include <vector>
#include <dlfcn.h>

#include "abi_contract.h"

namespace aivision
{
    namespace algo
    {

        /// 符号校验结果
        struct SymbolCheckResult
        {
            bool all_required_found = false;
            std::vector<std::string> missing_required;
            std::vector<std::string> optional_found;
            std::string error_message;
        };

        /// SoHandle — RAII 动态库句柄
        class SoHandle
        {
        public:
            /// 默认构造 (空句柄)
            SoHandle() = default;

            /// 加载 .so 文件并校验必需符号
            explicit SoHandle(const std::string &so_path);

            /// 析构时自动 dlclose
            ~SoHandle();

            /// 禁用拷贝，允许移动
            SoHandle(const SoHandle &) = delete;
            SoHandle &operator=(const SoHandle &) = delete;
            SoHandle(SoHandle &&other) noexcept;
            SoHandle &operator=(SoHandle &&other) noexcept;

            /// 检查是否已加载
            bool IsLoaded() const { return handle_ != nullptr; }

            /// 获取 .so 路径
            const std::string &GetPath() const { return so_path_; }

            /// 获取符号校验结果
            const SymbolCheckResult &GetCheckResult() const { return check_result_; }

            /// 通用 dlsym 查询 (返回函数指针)
            template <typename FuncType>
            FuncType GetSymbol(const std::string &name) const
            {
                if (!handle_)
                    return nullptr;
                return reinterpret_cast<FuncType>(dlsym(handle_, name.c_str()));
            }

            // ============================================================
            // 必需符号访问器
            // ============================================================

            algo_handle_t Init(const char *config_json)
            {
                auto fn = GetSymbol<decltype(&detector_init)>("detector_init");
                return fn ? fn(config_json) : nullptr;
            }

            int Infer(algo_handle_t handle, const hw_buffer_desc_t *input,
                      const char *context_json, infer_result_t *result)
            {
                auto fn = GetSymbol<decltype(&detector_infer)>("detector_infer");
                return fn ? fn(handle, input, context_json, result) : -1;
            }

            void Destroy(algo_handle_t handle)
            {
                auto fn = GetSymbol<decltype(&detector_destroy)>("detector_destroy");
                if (fn)
                    fn(handle);
            }

            // ============================================================
            // 可选符号访问器
            // ============================================================

            const char *Version()
            {
                auto fn = GetSymbol<decltype(&detector_version)>("detector_version");
                return fn ? fn() : nullptr;
            }

            const char *Name()
            {
                auto fn = GetSymbol<decltype(&detector_name)>("detector_name");
                return fn ? fn() : nullptr;
            }

            int SelfTest()
            {
                auto fn = GetSymbol<decltype(&detector_self_test)>("detector_self_test");
                return fn ? fn() : -1;
            }

            /// 手动释放 (关闭 .so)
            void Close();

        private:
            /// 校验所有必需符号
            SymbolCheckResult ValidateSymbols();

            void *handle_ = nullptr;
            std::string so_path_;
            SymbolCheckResult check_result_;
        };

        /// 动态库加载异常
        class SoLoadException : public std::runtime_error
        {
        public:
            explicit SoLoadException(const std::string &msg)
                : std::runtime_error(msg) {}
        };

    } // namespace algo
} // namespace aivision

#endif // AIVISION_ALGO_SO_HANDLE_H
