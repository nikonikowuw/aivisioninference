# ZLMediaKit Streaming Engine

```text
zlm/
├── docker-compose.yml    # Docker 容器化部署
├── config/
│   └── config.ini        # ZLM 配置文件（SIP、RTP、WebRTC、Webhook）
├── scripts/
│   └── deploy.sh          # 边缘设备本地裸机部署脚本
└── data/                  # 运行时数据（gitignored）
    ├── www/               # HTTP 静态资源 + 截图
    └── record/            # 录像文件
```

## 部署方式

### Docker（推荐）
```bash
docker compose -f zlm/docker-compose.yml up -d
```

### 本地裸机（边缘设备）
```bash
sudo bash zlm/scripts/deploy.sh
```

## 端口说明

| 端口 | 协议 | 用途 |
|------|------|------|
| 1935 | RTMP | 推流/拉流 |
| 554  | RTSP | 拉流 |
| 80   | HTTP | HTTP-FLV/HLS 播放、截图访问 |
| 5060 | UDP  | GB28181 SIP 信令 |
| 8000 | TCP  | WebRTC (RTC) |
| 9000 | TCP  | SRT 协议 |

> 采用 `network_mode: host`，无需手动映射端口。
