# 第一阶段：编译
FROM golang:1.23.5-alpine3.21 AS builder

WORKDIR /app

# 设置代理 (可选)
# ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 编译，去除符号表
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pilipili_backend main.go

# 第二阶段：运行
FROM alpine:3.21

WORKDIR /app

# 安装必要的系统库
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/pilipili_backend .

# 环境变量
ENV TZ=Asia/Shanghai

# 入口
ENTRYPOINT ["./pilipili_backend"]
CMD ["config.yaml"]
