// SoHandle 实现
#include "algo/so_handle.h"
#include <dlfcn.h>
#include <iostream>

namespace aivision
{
    namespace algo
    {

        SoHandle::SoHandle(const std::string &so_path)
            : so_path_(so_path)
        {
            handle_ = dlopen(so_path.c_str(), RTLD_NOW | RTLD_LOCAL);
            if (!handle_)
            {
                throw SoLoadException(
                    "Failed to load " + so_path + ": " + dlerror());
            }

            check_result_ = ValidateSymbols();
            if (!check_result_.all_required_found)
            {
                dlclose(handle_);
                handle_ = nullptr;
                std::string msg = "Missing required symbols in " + so_path + ": ";
                for (const auto &s : check_result_.missing_required)
                {
                    msg += s + " ";
                }
                throw SoLoadException(msg);
            }
        }

        SoHandle::~SoHandle()
        {
            Close();
        }

        SoHandle::SoHandle(SoHandle &&other) noexcept
            : handle_(other.handle_), so_path_(std::move(other.so_path_)), check_result_(std::move(other.check_result_))
        {
            other.handle_ = nullptr;
        }

        SoHandle &SoHandle::operator=(SoHandle &&other) noexcept
        {
            if (this != &other)
            {
                Close();
                handle_ = other.handle_;
                so_path_ = std::move(other.so_path_);
                check_result_ = std::move(other.check_result_);
                other.handle_ = nullptr;
            }
            return *this;
        }

        void SoHandle::Close()
        {
            if (handle_)
            {
                dlclose(handle_);
                handle_ = nullptr;
            }
        }

        SymbolCheckResult SoHandle::ValidateSymbols()
        {
            SymbolCheckResult result;
            const char *required_symbols[] = {
                "detector_init",
                "detector_infer",
                "detector_destroy",
            };

            result.all_required_found = true;
            for (const auto *sym : required_symbols)
            {
                if (!dlsym(handle_, sym))
                {
                    result.missing_required.push_back(sym);
                    result.all_required_found = false;
                }
            }

            // 检查可选符号
            const char *optional_symbols[] = {
                "detector_version",
                "detector_name",
                "detector_self_test",
            };
            for (const auto *sym : optional_symbols)
            {
                if (dlsym(handle_, sym))
                {
                    result.optional_found.push_back(sym);
                }
            }

            return result;
        }

    } // namespace algo
} // namespace aivision
