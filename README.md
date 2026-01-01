<h1 align="center">Go_Backend</h1>

<p align="center">一个实现 Emby 服务播放前后端分离的程序套件。</p>



![Commit Activity](https://img.shields.io/github/commit-activity/m/hsuyelin/PiliPili_Backend/main) ![Top Language](https://img.shields.io/github/languages/top/hsuyelin/PiliPili_Backend) ![Github License](https://img.shields.io/github/license/hsuyelin/PiliPili_Backend)



[English Version](https://github.com/hsuyelin/PiliPili_Backend/blob/main/README.md)

## 简介

1. 本项目是实现 Emby 媒体服务播放前后端分离的后端程序，需要与播放分离前端 [PiliPili Playback Frontend](https://github.com/hsuyelin/PiliPili_Frontend) 配套使用。
2. 本程序很大程度上基于 [YASS-Backend](https://github.com/FacMata/YASS-Backend)，并为了提高易用性和性能进行了优化。

------

## 原理

1. 使用特定的 `nginx` 配置（参考 [nginx.conf](https://github.com/hsuyelin/PiliPili_Backend/blob/main/nginx/nginx.conf)）监听指定端口，接收来自前端重定向的播放链接。

2. 解析播放链接中的 `path` 和 `signature`。

3. 解密 `signature` 以提取 `mediaId` 和 `expireAt`：

    - 如果解密成功，记录 `mediaId` 用于调试，并验证过期时间 (`expireAt`)。如果有效，则认证通过；否则返回 `401 Unauthorized` 错误。
    - 如果解密失败，立即返回 `401 Unauthorized` 错误。

4. **路径清洗（Sanitize）**：对解析出的 `path` 进行清洗（例如处理 `..`）以确保安全性，防止目录遍历攻击，然后将其与配置文件中的 `StorageBasePath` 组合，生成绝对本地文件路径。

5. 获取文件信息：

    - 如果文件不存在或是一个目录，返回相应的错误（404 Not Found 或 403 Forbidden）。
    - 如果获取失败，返回 `500 Internal Server Error`。

6. **零拷贝流式传输（Zero-Copy Streaming）**：

    - 利用系统的 `sendfile` 机制（通过 Go 标准库 `http.ServeFile`）将数据直接从磁盘传输到网卡。
    - 该机制自动处理 `Content-Range` 请求（用于断点续传）和 MIME 类型检测，显著降低 CPU 使用率并大幅提高吞吐量。

    ![sequenceDiagram](https://github.com/hsuyelin/PiliPili_Backend/blob/main/img/sequenceDiagram_CN.png)

------

## 功能

- 兼容所有版本的 Emby 服务器。
- **高性能**：使用零拷贝技术 (`sendfile`) 以最小化 CPU 使用率并最大化吞吐量。
- **安全**：实现了路径清洗以防止目录遍历攻击。
- 支持高并发请求。
- 支持签名解密并拦截过期的播放链接。

------

## 配置文件

```yaml
# Configuration for PiliPili Backend

# LogLevel defines the level of logging (e.g., INFO, DEBUG, ERROR)
LogLevel: "INFO"

# EncryptionKey is used for encryption and obfuscation of data.
Encipher: "vPQC5LWCN2CW2opz"

# StorageBasePath is the base directory where files are stored. This is a prefix for the storage paths.
StorageBasePath: "/mnt/anime/"

# Server configuration
Server:
  port: "60002"  # Port on which the server will listen
