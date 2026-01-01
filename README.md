<h1 align="center">Go_Backend</h1>

<p align="center">一个实现 Emby 服务播放前后端分离的后端程序套件。</p>

[English Version](https://github.com/hsuyelin/PiliPili_Backend/blob/main/README.md)

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

## How to Use

### 1. Install Using Docker (Recommended)

#### 1.1 Create a Docker Directory

```shell
mkdir -p /data/docker/go_backend
```

#### 1.2 Create Configuration Folder and File

```shell
cd /data/docker/go_backend
mkdir -p config && cd config
```

Copy [config.yaml](https://github.com/Moxi007/Go_Backend/edit/main/config.yaml) to the `config` folder and edit it as needed.

#### 1.3 Create docker-compose.yaml

Navigate back to the `/data/docker/pilipili_backend` directory, and copy [docker-compose.yml](https://github.com/Moxi007/Go_Backend/edit/main/docker/docker-compose.yml) to this directory.

#### 1.4 Start the Container

```shell
docker-compose pull && docker-compose up -d
```

### 2. Manual Installation

#### 2.1 Install the Go Environment

##### 2.1.1 Remove Existing Go Installation

Forcefully remove any existing Go installation to ensure version compatibility.

```shell
rm -rf /usr/local/go
```

##### 2.1.2 Download and Install the Latest Version of Go

```shell
wget -q -O /tmp/go.tar.gz https://go.dev/dl/go1.23.5.linux-amd64.tar.gz && tar -C /usr/local -xzf /tmp/go.tar.gz && rm /tmp/go.tar.gz
```

##### 2.1.3 Add Go to the Environment Variables

```shell
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
```

##### 2.1.4 Verify Installation

```shell
go version # If the output is "go version go1.23.5 linux/amd64," the installation was successful.
```

#### 2.2 Clone the Backend Program to Local Machine

For example, to clone it to the `/data/emby_backend` directory:

```shell
git clone https://github.com/Moxi007/Go_Backend.git /data/emby_backend
```

#### 2.3 Enter the Backend Program Directory and Edit the Configuration File

```yaml
# Configuration for Go Backend

# LogLevel defines the level of logging (e.g., INFO, DEBUG, ERROR)
LogLevel: "INFO"

# EncryptionKey is used for encryption and obfuscation of data.
Encipher: "vPQC5LWCN2CW2opz"

# StorageBasePath is the base directory where files are stored. This is a prefix for the storage paths.
StorageBasePath: "/mnt/anime/"

# Server configuration
Server:
  port: "60002"  # Port on which the server will listen
```

#### 2.4 Run the Program

```shell
nohup go run main.go config.yaml > streamer.log 2>&1 &
```

