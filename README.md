<h1 align="center">Go_Backend</h1>

<p align="center">A program suite for separating the frontend and backend of Emby service playback.</p>



![Commit Activity](https://img.shields.io/github/commit-activity/m/hsuyelin/PiliPili_Backend/main) ![Top Language](https://img.shields.io/github/languages/top/hsuyelin/PiliPili_Backend) ![Github License](https://img.shields.io/github/license/hsuyelin/PiliPili_Backend)



[中文版本](https://github.com/hsuyelin/PiliPili_Backend/blob/main/README_CN.md)

## Introduction

1. This project is the backend application for separating Emby media service playback into frontend and backend components. It is designed to work with the playback frontend [PiliPili Playback Frontend](https://github.com/hsuyelin/PiliPili_Frontend).
2. This program is largely based on [YASS-Backend](https://github.com/FacMata/YASS-Backend), with optimizations made for improved usability and performance.

------

## Principle

1. Use a specific `nginx`configuration (refer to [nginx.conf](https://github.com/hsuyelin/PiliPili_Backend/blob/main/nginx/nginx.conf) to listen on a designated port for redirected playback links from the frontend.

2. Parse the `path` and `signature` from the playback link.

3. Decrypt the `signature` to extract `mediaId`and `expireAt`:

    - If decryption succeeds, log the `mediaId` for debugging and validate the expiration time (`expireAt`). If valid, authentication passes; otherwise, return a `401 Unauthorized` error.
    - If decryption fails, immediately return a `401 Unauthorized` error.

4. **Sanitize** the parsed `path` (e.g., resolving `..`) to ensure security, then combine it with the `StorageBasePath` from the configuration file to generate the absolute local file path.

5. Retrieve file information:

    - If the file does not exist or is a directory, return an appropriate error (404 Not Found or 403 Forbidden).
    - If retrieval fails, return a `500 Internal Server Error`.

6. **Zero-Copy Streaming**:

    - Use the system's `sendfile` mechanism (via `http.ServeFile`) to transfer data directly from disk to the network card.
    - This automatically handles `Content-Range` requests (for break-point resumption) and MIME type detection, significantly reducing CPU usage and increasing throughput.

    ![sequenceDiagram](https://github.com/hsuyelin/PiliPili_Backend/blob/main/img/sequenceDiagram.png)

------

## Features

- Compatible with all Emby server versions.
- **High Performance**: Uses zero-copy technology (`sendfile`) to minimize CPU usage and maximize throughput.
- **Secure**: Implements path sanitization to prevent directory traversal attacks.
- Supports high-concurrency requests.
- Supports signature decryption and blocks expired playback links.

------

## Configuration File

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
