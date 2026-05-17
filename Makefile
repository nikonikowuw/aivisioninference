.PHONY: help docker-build docker-up docker-down docker-logs docker-rebuild docker-prod docker-prod-down docker-migrate docker-prod-migrate init

# 默认目标
help:
	@echo ""
	@echo "  Docker 部署命令"
	@echo "  =================="
	@echo ""
	@echo "  make init              一键初始化（创建 .env + 启动服务 + 迁移）"
	@echo "  make docker-up         启动开发环境"
	@echo "  make docker-down       停止所有服务"
	@echo "  make docker-logs       查看服务日志"
	@echo "  make docker-rebuild    重新构建并启动"
	@echo ""
	@echo "  生产环境"
	@echo "  =================="
	@echo "  make docker-prod       启动生产环境"
	@echo "  make docker-prod-down  停止生产环境"
	@echo "  make docker-migrate    运行数据库迁移"
	@echo ""

## init: 一键初始化
init:
	@cp -n .env.example .env 2>/dev/null || true
	@echo "Starting services..."
	docker-compose up -d
	@echo "Waiting for database..."
	@until docker-compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Running migrations..."
	docker-compose exec server ./niko-admin-migrate
	@echo ""
	@echo "✓ 初始化完成！访问 http://localhost:8080"
	@echo ""

## docker-build: 构建 Docker 镜像
docker-build:
	docker-compose build

## docker-up: 启动开发环境
docker-up:
	docker-compose up -d

## docker-down: 停止所有服务
docker-down:
	docker-compose down

## docker-logs: 查看服务日志
docker-logs:
	docker-compose logs -f server

## docker-rebuild: 重新构建并启动
docker-rebuild:
	docker-compose up -d --build

## docker-prod: 启动生产环境
docker-prod:
	@test -f .env.production || (echo "错误: 请先创建 .env.production 文件" && exit 1)
	docker-compose -f docker-compose.prod.yml --env-file .env.production up -d --build

## docker-prod-down: 停止生产环境
docker-prod-down:
	docker-compose -f docker-compose.prod.yml down

## docker-migrate: 运行数据库迁移
docker-migrate:
	docker-compose exec server ./niko-admin-migrate

## docker-prod-migrate: 运行生产环境数据库迁移
docker-prod-migrate:
	docker-compose -f docker-compose.prod.yml exec server ./niko-admin-migrate
