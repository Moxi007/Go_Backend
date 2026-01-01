<h1 align="center">Go_Backend</h1>

<p align="center">一个实现 Emby 服务播放前后端分离的后端程序套件。</p>

## 简介

1. 本项目是实现 Emby 媒体服务播放前后端分离的后端程序，需要与播放分离前端 [PiliPili Playback Frontend](https://github.com/Moxi007/PiliPili_Frontend) 配套使用。
2. 本程序很大程度上基于 [YASS-Backend](https://github.com/FacMata/YASS-Backend)，并为了提高易用性和极致的传输性能进行了大幅重构和优化。

------

## 原理

1. **流量劫持**：使用特定的 `nginx` 配置（参考 [nginx.conf](https://github.com/Moxi007/Go_Backend/blob/main/nginx/nginx.conf)）监听指定端口，接收来自前端重定向过来的播放链接。

2. **参数解析**：程序从请求中解析出 `path`（文件相对路径）和 `signature`（加密签名）。

3. **安全鉴权**：
    - 解密 `signature` 以提取 `mediaId` 和 `expireAt`。
    - 验证签名是否合法以及链接是否过期。如果验证失败，立即返回 `401 Unauthorized`。

4. **路径清洗与映射**：
    - **安全清洗**：对解析出的 `path` 进行深度清洗（例如处理 `..` 路径穿越符），防止恶意读取服务器敏感文件。
    - **路径映射**：将清洗后的路径与配置文件中的 `StorageBasePath` 组合，生成文件的绝对本地路径。

5. **零拷贝流式传输 (Zero-Copy Streaming)**：
    - 利用操作系统的 `sendfile` 机制（通过 Go 标准库 `http.ServeFile` 实现）。
    - 数据直接从磁盘缓存（Page Cache）传输到网卡，**无需经过用户态内存拷贝**。
    - 自动处理 `Content-Range`（断点续传）、MIME 类型检测和缓存控制，极大降低 CPU 占用并跑满网络带宽。

------

## 功能特点

- **全版本兼容**：支持所有版本的 Emby 服务器。
- **极致性能**：采用零拷贝技术 (`sendfile`)，CPU 占用极低，适合在大流量高并发场景下使用。
- **安全加固**：
    - 内置路径遍历防御机制。
    - 签名加密与过期时间校验，有效防止盗链。
- **高并发**：优化的 Go 协程模型，轻松处理数千并发连接。
- **极小体积**：基于 Alpine 的 Docker 镜像，体积优化至约 20MB。

------

## 配置文件

配置文件默认路径：`config.yaml`

```yaml
# Go Backend 配置文件

# LogLevel 定义日志级别 (可选: DEBUG, INFO, WARN, ERROR)
LogLevel: "INFO"

# EncryptionKey 用于数据的加密和混淆，必须与前端保持一致
Encipher: "vPQC5LWCN2CW2opz"

# StorageBasePath 是本地文件存储的基础目录，将作为前端传来的相对路径的前缀
StorageBasePath: "/mnt/anime/"


# 服务器配置
Server:
  port: 60002  # 服务器监听的端口

```
------

## 如何使用

### 步骤 0: Nginx 配置 (前置条件)

- 后端程序默认监听 60002 端口，建议配合 Nginx 使用 HTTPS 并进行反向代理。

- 找到你的 Nginx 配置文件。

- 添加如下配置，将访问 /stream 的流量转发给后端。

 ```shell
server {
    listen 443 ssl;
    server_name streamer.example.com; # 你的推流域名

    # SSL 证书配置...
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # 核心转发配置
    location /stream {
        proxy_pass [http://127.0.0.1:60002](http://127.0.0.1:60002);
        
        # 传递真实 IP
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 禁用缓冲，这对流媒体非常重要
        proxy_buffering off;
        proxy_request_buffering off;
        
        # 长连接设置
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

### 方式 1: Docker 安装 (推荐)

#### 1.1 创建目录

```shell
mkdir -p /data/docker/go_backend
```

#### 1.2 创建配置文件

```shell
cd /data/docker/go_backend
mkdir -p config && cd config
```

将 config.yaml 复制到 config 文件夹中，并根据实际情况编辑（特别是 StorageBasePath 和 Encipher）。

#### 1.3 创建 docker-compose.yaml

返回 /data/docker/go_backend 目录，创建 docker-compose.yml

#### 1.4 启动容器

```shell
docker-compose pull && docker-compose up -d
```

### 方式 2: 手动编译安装

#### 2.1 安装 Go 环境

```shell
# 1. 下载 Go (请根据最新版本调整 URL)
wget -q -O /tmp/go.tar.gz [https://go.dev/dl/go1.23.5.linux-amd64.tar.gz](https://go.dev/dl/go1.23.5.linux-amd64.tar.gz) 

# 2. 解压安装
rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz && rm /tmp/go.tar.gz

# 3. 配置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc

# 4. 验证安装
go version
```

##### 2.2 下载代码

```shell
git clone [https://github.com/Moxi007/Go_Backend.git](https://github.com/Moxi007/Go_Backend.git) /data/emby_backend
cd /data/emby_backend
```

##### 2.3 编译与配置

```shell
vi config.yaml
```
编译二进制文件（推荐使用 build 生成二进制文件，比 go run 启动更快且更稳定）：

```shell
# -s -w 参数用于去除调试符号，减小体积
go build -ldflags="-s -w" -o pilipili_backend main.go
```

##### 2.4 运行程序

```shell
# 前台运行测试
./pilipili_backend config.yaml

# 后台运行 (使用 nohup)
nohup ./pilipili_backend config.yaml > streamer.log 2>&1 &
```
