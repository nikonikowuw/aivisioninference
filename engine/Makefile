# aivision-engine Makefile
# 用法: make [target]

# ============================================================
# 变量
# ============================================================
BUILD_DIR       ?= build
RK_BUILD_DIR    ?= build-rk3568
RK_CROSS_DIR    ?= build-rk3568-cross
CMAKE_BUILD_TYPE ?= Debug
JOBS            ?= $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
SED_INPLACE     := $(shell if sed --version >/dev/null 2>&1; then echo "sed -i"; else echo "sed -i ''"; fi)
CMAKE_POLICY_ARGS ?=

# ============================================================
# 默认目标: 开发模式 (x86_64, stub HAL)
# ============================================================
.PHONY: all
all: dev

# ============================================================
# 开发模式构建 (不启用 RKMPP)
# ============================================================
.PHONY: dev
dev: flatbuf
	@mkdir -p $(BUILD_DIR)
	cmake -B $(BUILD_DIR) -DAIVISION_WITH_RKMPP=OFF -DCMAKE_BUILD_TYPE=$(CMAKE_BUILD_TYPE) $(CMAKE_POLICY_ARGS)
	cmake --build $(BUILD_DIR) -j$(JOBS)
	@echo ""
	@echo "✅ 开发构建完成"
	@echo "   引擎: $(BUILD_DIR)/aivision-engine"
	@echo "   HAL:  $(BUILD_DIR)/libaivision-hal-rkmpp.so"

# ============================================================
# Release 构建 (x86_64, stub HAL)
# ============================================================
.PHONY: release
release: flatbuf
	@mkdir -p $(BUILD_DIR)
	cmake -B $(BUILD_DIR) -DAIVISION_WITH_RKMPP=OFF -DCMAKE_BUILD_TYPE=Release $(CMAKE_POLICY_ARGS)
	cmake --build $(BUILD_DIR) -j$(JOBS)
	@echo ""
	@echo "✅ Release 构建完成"

# ============================================================
# RK3568 本地编译 (在 RK3568 目标机上执行)
# ============================================================
.PHONY: rk3568
rk3568:
	@mkdir -p $(RK_BUILD_DIR)
	cmake -B $(RK_BUILD_DIR) -DAIVISION_WITH_RKMPP=ON -DCMAKE_BUILD_TYPE=Release $(CMAKE_POLICY_ARGS)
	cmake --build $(RK_BUILD_DIR) -j$(JOBS)
	@echo ""
	@echo "✅ RK3568 构建完成"
	@echo "   引擎: $(RK_BUILD_DIR)/aivision-engine"
	@echo "   HAL:  $(RK_BUILD_DIR)/libaivision-hal-rkmpp.so"

# ============================================================
# RK3568 交叉编译
# 用法: make rk3568-cross SYSROOT=/path/to/sysroot
# ============================================================
.PHONY: rk3568-cross
rk3568-cross:
ifndef SYSROOT
	$(error SYSROOT 未设置，用法: make rk3568-cross SYSROOT=/path/to/rk3568/sysroot)
endif
	@mkdir -p $(RK_CROSS_DIR)
	cmake -B $(RK_CROSS_DIR) \
		-DCMAKE_TOOLCHAIN_FILE=cmake/rk3568-toolchain.cmake \
		-DRK_SDK_PATH=$(SYSROOT) \
		-DAIVISION_WITH_RKMPP=ON \
		-DCMAKE_BUILD_TYPE=Release \
		$(CMAKE_POLICY_ARGS)
	cmake --build $(RK_CROSS_DIR) -j$(JOBS)
	@echo ""
	@echo "✅ RK3568 交叉编译完成"
	@echo "   HAL: $(RK_CROSS_DIR)/libaivision-hal-rkmpp.so"

# ============================================================
# 测试
# ============================================================
.PHONY: test
test:
	@mkdir -p $(BUILD_DIR)
	cmake -B $(BUILD_DIR) -DAIVISION_WITH_RKMPP=OFF -DBUILD_TESTING=ON $(CMAKE_POLICY_ARGS)
	cmake --build $(BUILD_DIR) --target test_rkmpp_pipeline -j$(JOBS)
	cd $(BUILD_DIR) && ctest --output-on-failure
	@echo ""
	@echo "✅ 测试完成"

# ============================================================
# 安装 (默认到 /usr/local)
# ============================================================
.PHONY: install
install:
	cmake --install $(BUILD_DIR)
	@echo ""
	@echo "✅ 安装完成"

# ============================================================
# 清理
# ============================================================
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) $(RK_BUILD_DIR) $(RK_CROSS_DIR)
	@echo "✅ 已清理所有构建目录"

.PHONY: clean-dev
clean-dev:
	rm -rf $(BUILD_DIR)
	@echo "✅ 已清理开发构建目录"

.PHONY: clean-rk
clean-rk:
	rm -rf $(RK_BUILD_DIR) $(RK_CROSS_DIR)
	@echo "✅ 已清理 RK3568 构建目录"

# ============================================================
# macOS 构建 (开发模式)
# ============================================================
.PHONY: macos
macos:
	@mkdir -p $(BUILD_DIR)
	cmake -B $(BUILD_DIR) -DAIVISION_WITH_RKMPP=OFF -DCMAKE_BUILD_TYPE=$(CMAKE_BUILD_TYPE) $(CMAKE_POLICY_ARGS)
	cmake --build $(BUILD_DIR) -j$(JOBS)
	@echo ""
	@echo "✅ macOS 构建完成"

# ============================================================
# FlatBuffers 代码生成
# ============================================================
.PHONY: flatbuf
flatbuf:
	@echo "生成 FlatBuffers 头文件..."
	mkdir -p include/proto/flatbuf
	flatc --cpp --gen-mutable --gen-object-api \
		-I ../proto/flatbuf \
		-o include/proto/flatbuf \
		../proto/flatbuf/common.fbs \
		../proto/flatbuf/envelope.fbs \
		../proto/flatbuf/commands.fbs \
		../proto/flatbuf/results.fbs
	@echo "修复 include 路径..."
	@cd include/proto/flatbuf && \
		for f in *_generated.h; do \
			$(SED_INPLACE) 's|#include "common_generated.h"|#include "proto/flatbuf/common_generated.h"|g' "$$f"; \
			$(SED_INPLACE) 's|#include "envelope_generated.h"|#include "proto/flatbuf/envelope_generated.h"|g' "$$f"; \
			$(SED_INPLACE) 's|#include "commands_generated.h"|#include "proto/flatbuf/commands_generated.h"|g' "$$f"; \
			$(SED_INPLACE) 's|#include "results_generated.h"|#include "proto/flatbuf/results_generated.h"|g' "$$f"; \
		done
	@echo "✅ FlatBuffers 头文件已生成到 engine/include/proto/flatbuf/"
# ============================================================
.PHONY: help
help:
	@echo "aivision-engine 构建系统"
	@echo ""
	@echo "用法: make [target]"
	@echo ""
	@echo "目标:"
	@echo "  all (默认)    开发模式构建 (x86_64, stub HAL)"
	@echo "  dev           同 all"
	@echo "  release       Release 构建 (x86_64, stub HAL)"
	@echo "  rk3568        RK3568 本地编译 (在目标机上执行)"
	@echo "  rk3568-cross  RK3568 交叉编译 (需设置 SYSROOT)"
	@echo "  macos         macOS 开发构建"
	@echo "  flatbuf       生成 FlatBuffers 头文件"
	@echo "  test          编译并运行测试"
	@echo "  install       安装到系统目录"
	@echo "  clean         清理所有构建目录"
	@echo "  clean-dev     仅清理开发构建目录"
	@echo "  clean-rk      仅清理 RK3568 构建目录"
	@echo "  help          显示此帮助"
	@echo ""
	@echo "变量:"
	@echo "  BUILD_DIR     开发构建目录 (默认: build)"
	@echo "  RK_BUILD_DIR  RK3568 构建目录 (默认: build-rk3568)"
	@echo "  JOBS          并行编译数 (默认: $(JOBS))"
	@echo "  CMAKE_BUILD_TYPE  开发构建类型 (默认: $(CMAKE_BUILD_TYPE))"
	@echo "  SYSROOT       交叉编译 sysroot 路径"
	@echo ""
	@echo "示例:"
	@echo "  make                    # 开发构建"
	@echo "  make release            # Release 构建"
	@echo "  make test               # 运行测试"
	@echo "  make rk3568             # 在 RK3568 上编译"
	@echo "  make rk3568-cross SYSROOT=~/rk3568-sdk  # 交叉编译"
	@echo "  make clean              # 清理"
