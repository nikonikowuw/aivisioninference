APP_NAME      ?= niko-admin
BUILD_DIR      = ./bin
MAIN_PATH      = ./cmd/server
ENV           ?= dev
GO_FLAGS      ?= -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(shell date +%Y%m%d%H%M%S)"
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: serve run dev test deploy build unit-test lint clean help migrate

## serve: 开发模式 — 启动依赖 + 热重载（需要 Docker）
serve: docker-up
	@echo "Starting development server with air..."
	air

## dev: 本地热重载开发（无需 Docker，需自行启动 postgres+redis）
dev:
	@echo "Starting development server with air (local)..."
	air

# 空操作，使命令行参数 test/deploy 不报错
test:
deploy:

## run: 本地运行（make run / make run dev / make run deploy，默认 dev）
run:
	NIKO_APP_ENV=$(if $(filter-out $@,$(MAKECMDGOALS)),$(filter-out $@,$(MAKECMDGOALS)),$(ENV)) go run $(MAIN_PATH)

## build: 编译当前平台二进制
build:
	@mkdir -p $(BUILD_DIR)
	go build $(GO_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)"

## build-linux: 交叉编译 Linux amd64
build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)-linux-amd64"

## swag: 生成 Swagger API 文档
swag:
	swag init -g internal/router/router.go -o docs/

## gen: 代码生成（CRUD）
gen:
	go run cmd/gen/main.go --all

## unit-test: 运行测试
unit-test:
	go test -v -race -coverprofile=coverage.out ./...

## lint: 代码规范检查
lint:
	golangci-lint run ./...

## migrate: 执行数据库迁移
migrate:
	go run cmd/migrate/main.go

## docker-build: 构建 Docker 镜像
docker-build:
	docker build -t $(APP_NAME):$(VERSION) .

## docker-up: 启动 Docker 服务
docker-up:
	cd .. && docker-compose up -d postgres redis

## docker-down: 停止 Docker 服务
docker-down:
	cd .. && docker-compose down

## clean: 清理构建产物
clean:
	rm -rf $(BUILD_DIR) tmp/ coverage.out

## init: 一键初始化
init:
	cp -n .env.example .env 2>/dev/null || true
	cd .. && docker-compose up -d postgres redis
	@echo "Waiting for database..."
	@sleep 3
	go mod tidy
	go run cmd/migrate/main.go
	@echo "--- Init complete! Run 'make serve' to start ---"

## help: 显示帮助信息
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## [a-zA-Z_-]+:.*' $(MAKEFILE_LIST) | \
		sed 's/^## //' | \
		awk 'BEGIN {FS = ": "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
