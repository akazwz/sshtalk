FROM golang:1.24.2-alpine AS builder

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -o sshtalk .

# 使用更小的镜像作为最终镜像
FROM alpine:3.21

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates

# 从构建阶段复制编译好的应用
COPY --from=builder /app/sshtalk /app/

# 创建.ssh目录
RUN mkdir -p /app/.ssh && chmod 700 /app/.ssh

EXPOSE $PORT

ENTRYPOINT ["/app/sshtalk"]
