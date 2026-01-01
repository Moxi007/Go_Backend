# 第一阶段：构建阶段
FROM golang:1.23.5-alpine3.21 AS builder

WORKDIR /app

# 【优化1】取消注释并启用国内代理，确保构建速度和成功率
# ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 【优化2】新增此行：自动整理依赖
# 自动下载代码中引用但 go.mod 缺失的库，防止构建报错
RUN go mod tidy

# -s -w 去除符号表和调试信息，减小二进制体积
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o go_backend main.go

# 第二阶段：运行阶段
FROM alpine:3.21

WORKDIR /app

# 安装基础库（SSL证书、时区数据）
RUN apk --no-cache add ca-certificates tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /app/go_backend .

# 设置时区
ENV TZ=Asia/Shanghai

# 容器启动命令
ENTRYPOINT ["./go_backend"]
CMD ["config.yaml"]
