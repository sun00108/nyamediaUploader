# nyaupload

`nyaupload` 是一个命令行工具，用于通过 `nyamedia-bot` 将已请求的媒体文件上传到 OneDrive。

它支持：

- 基于 Telegram 的 CLI 登录
- 本地上传请求缓存
- 交互式选择上传目标
- 使用 OneDrive upload session 进行大文件上传
- 通过 GitHub Actions 构建多平台发布产物

## 安装

### 下载发布版二进制

从 GitHub Releases 下载对应平台的压缩包：

- `nyaupload-darwin-arm64.zip`
- `nyaupload-darwin-amd64.zip`
- `nyaupload-linux-amd64.zip`
- `nyaupload-linux-armv7.zip`
- `nyaupload-windows-amd64.zip`

每个 zip 中只包含一个二进制文件：

- `nyaupload`
- Windows 下为 `nyaupload.exe`

## 使用方法

### 1. 登录

```bash
nyaupload login
```

CLI 会：

1. 输出一个登录链接
2. 等待你在浏览器中完成 Telegram 登录
3. 提示你粘贴授权码
4. 将返回的会话信息保存到本地

本地 session 文件：

- `~/.config/nyaupload/session.json`

### 2. 创建上传请求

电影或非剧集请求：

```bash
nyaupload request --request-id 123
```

剧集请求：

```bash
nyaupload request --request-id 123 --season 1 --episode 2
```

该命令会调用 bot，并把本地请求记录写入：

- `~/.config/nyaupload/upload_requests.json`

### 3. 上传文件

```bash
nyaupload upload /path/to/file.mkv
```

CLI 会：

1. 读取本地上传请求列表
2. 显示交互式选择界面
3. 让你选择正确的媒体条目
4. 向 bot 请求 OneDrive upload session
5. 分片上传文件
6. 在上传完成后通知 bot
7. 从本地缓存中删除已使用的请求

可选参数：

```bash
nyaupload upload /path/to/file.mkv
```

### 4. 登出

```bash
nyaupload logout
```

该命令会删除本地 session 文件。

## 文件命名规则

执行 `nyaupload upload` 时，上传后的文件名不会直接使用本地文件名，而是根据请求元数据生成。

规则：

- 电影或非剧集：
  `{title}.{ext}`
- 剧集：
  `{title} - S{season}E{episode}.{ext}`

示例：

- `Inception.mkv`
- `Frieren - S01E03.mkv`

## 本地文件

默认本地文件：

- `~/.config/nyaupload/session.json`
- `~/.config/nyaupload/upload_requests.json`

## 命令汇总

```bash
nyaupload login
nyaupload logout
nyaupload request --request-id 123 [--season 1 --episode 2]
nyaupload upload /path/to/file.mkv
```
