# 第一阶段：构建阶段
FROM golang:1.23.5-alpine3.21 AS builder

WORKDIR /app

# 设置 Go 代理（可选，国内环境建议开启）
# ENV GOPROXY=https://goproxy.cn,direct

# 预下载依赖，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
# -s -w 去除符号表和调试信息，减小二进制体积
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pilipili_backend main.go

# 第二阶段：运行阶段
FROM alpine:3.21

WORKDIR /app

# 安装基础库（SSL证书、时区数据）
RUN apk --no-cache add ca-certificates tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /app/pilipili_backend .

# 设置时区
ENV TZ=Asia/Shanghai

# 容器启动命令
ENTRYPOINT ["./pilipili_backend"]
CMD ["config.yaml"]
