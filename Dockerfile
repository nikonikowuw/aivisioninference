# ==================== Stage 1: Build Frontend ====================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# 安装依赖
COPY web/package.json web/package-lock.json ./
RUN npm ci --legacy-peer-deps

# 构建前端
COPY web/ ./
RUN npm run build

# ==================== Stage 2: Build Backend ====================
FROM golang:1.26-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /app

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o niko-admin ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o niko-admin-migrate ./cmd/migrate

# ==================== Stage 3: Runtime ====================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 复制后端二进制
COPY --from=backend-builder /app/niko-admin .
COPY --from=backend-builder /app/niko-admin-migrate .
COPY --from=backend-builder /app/configs ./configs

# 复制前端静态文件
COPY --from=frontend-builder /app/web/dist ./web/dist

# 创建必要的目录
RUN mkdir -p logs uploads

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

EXPOSE 8080

CMD ["./niko-admin"]
