# StarData 多阶段构建 Dockerfile
# 用法:
#   构建镜像: docker build -t stardata:latest -f deploy/Dockerfile .
#   运行:     docker run -p 9080:9080 stardata:latest start /project
#
# 如需使用本机 DuckDB 数据: 挂载 /data 卷
#
# 注: web-local 已移除, CLI 不再嵌入前端; 企业端前端由 deploy/Dockerfile.web-admin 独立构建托管。
#
# Stage 1: Go 构建
FROM golang:1.26-alpine AS builder

WORKDIR /build

# 安装系统依赖 (proto 生成等)
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 嵌入 DuckDB 扩展
RUN go run scripts/embed_duckdb_ext/main.go

# 构建静态链接的 stardata 二进制
# -ldflags: 压缩体积, 移除调试信息
# CGO_ENABLED=0: 静态链接, 适合 scratch 镜像
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.Version=$(date -u +%Y%m%d)-dev" \
    -o stardata \
    cli/main.go

# Stage 2: 运行镜像
FROM alpine:3.20

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1001 stardata && \
    adduser -D -u 1001 -G stardata stardata

# 复制二进制
COPY --from=builder /build/stardata /usr/local/bin/stardata

# 创建必要的数据目录
RUN mkdir -p /data/stardata /data/storage && \
    chown -R stardata:stardata /data

USER stardata

# 安装 DuckDB 扩展
RUN stardata runtime install-duckdb-extensions

EXPOSE 9080 9090

ENTRYPOINT ["stardata"]
CMD ["start"]
