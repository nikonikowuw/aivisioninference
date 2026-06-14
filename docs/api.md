# API 参考

本文档列出 AIVisionInference 平台所有外部 API 接口，按功能模块组织。

---

## 边缘节点管理

所有边缘节点管理接口均需 **管理员权限**，在请求头携带 JWT Token：

```
Authorization: Bearer <admin-jwt-token>
```

### 创建边缘节点

```
POST /api/v1/edge-nodes
```

**请求体：**

```json
{
  "name": "edge-node-01",
  "description": "车间东北角推理节点",
  "endpoint": "http://192.168.1.100:8080",
  "ipc_addr": "192.168.1.100:9500",
  "max_load": 4,
  "remark": "位置：3 号产线"
}
```

| 字段         | 类型   | 必填 | 说明                     |
|--------------|--------|------|--------------------------|
| name         | string | 是   | 节点名称，1-128 字符     |
| description  | string | 否   | 描述，最多 500 字符      |
| endpoint     | string | 是   | 引擎 HTTP 地址，URL 格式 |
| ipc_addr     | string | 是   | 引擎 IPC 地址            |
| max_load     | int    | 是   | 最大并行任务数，>= 1     |
| remark       | string | 否   | 备注，最多 1000 字符     |

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid-string",
    "name": "edge-node-01",
    "endpoint": "http://192.168.1.100:8080",
    "ipc_addr": "192.168.1.100:9500",
    "max_load": 4,
    "status": "offline",
    "enabled": true,
    "auth_token": "eyJhbGciOiJIUzI1NiIs...",
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "created_at": "2026-06-14T10:00:00+08:00"
  }
}
```

> **注意**：`auth_token` / `token` 仅在创建时返回一次，请立即复制并配置到引擎。Token 有效期 10 年。

---

### 查询节点列表

```
GET /api/v1/edge-nodes?page=1&page_size=20&keyword=&status=
```

**查询参数：**

| 参数      | 类型   | 必填 | 说明                                     |
|-----------|--------|------|------------------------------------------|
| page      | int    | 否   | 页码，默认 1                             |
| page_size | int    | 否   | 每页数量，默认 20                        |
| keyword   | string | 否   | 关键词搜索（匹配名称、描述、端点等）     |
| status    | string | 否   | 状态筛选：`online` / `offline` / `error` / `disabled` |

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "uuid-string",
        "name": "edge-node-01",
        "status": "online",
        "current_load": 2,
        "max_load": 4,
        "endpoint": "http://192.168.1.100:8080",
        "hal_platform": "macos",
        "engine_version": "1.0.0",
        "last_heartbeat": "2026-06-14T10:00:05+08:00"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 查询节点详情

```
GET /api/v1/edge-nodes/:id
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid-string",
    "name": "edge-node-01",
    "status": "online",
    "endpoint": "http://192.168.1.100:8080",
    "ipc_addr": "192.168.1.100:9500",
    "current_load": 2,
    "max_load": 4,
    "hal_platform": "macos",
    "engine_version": "1.0.0",
    "uptime": 3600,
    "enabled": true,
    "hardware_info": {
      "cpu_model": "Apple M1 Pro",
      "gpu_model": "Apple M1 GPU",
      "total_memory": 17179869184,
      "platform": "macos"
    },
    "last_heartbeat": "2026-06-14T10:00:05+08:00",
    "description": "车间东北角推理节点",
    "remark": "位置：3 号产线",
    "created_at": "2026-06-14T10:00:00+08:00"
  }
}
```

---

### 更新节点

```
PUT /api/v1/edge-nodes/:id
```

**请求体（所有字段可选）：**

```json
{
  "name": "edge-node-01-updated",
  "description": "更新后的描述",
  "endpoint": "http://192.168.1.101:8080",
  "ipc_addr": "192.168.1.101:9500",
  "max_load": 8,
  "enabled": true,
  "status": "online",
  "remark": "已迁移至 4 号产线"
}
```

---

### 删除节点（软删除）

```
DELETE /api/v1/edge-nodes/:id
```

**响应：**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 引擎心跳上报

```
POST /api/v1/edge-nodes/:id/heartbeat
```

> 此接口使用 **节点 JWT Token** （非管理员 Token）鉴权，在请求头携带：
>
> ```
> Authorization: Bearer <node-jwt-token>
> ```

**请求体：**

```json
{
  "uptime": 3600,
  "current_load": 2,
  "cpu_usage": 45.5,
  "memory_usage": 62.3,
  "engine_version": "1.0.0",
  "hardware_info": {
    "cpu_model": "Apple M1 Pro",
    "gpu_model": "Apple M1 GPU",
    "total_memory": 17179869184,
    "platform": "macos"
  },
  "installed_algorithms": [
    {
      "algo_package_id": "pkg-face-recognition",
      "algo_name": "face_recognition",
      "version": "1.0.0",
      "status": "installed",
      "installed_at": "2026-06-14T09:00:00+08:00"
    }
  ]
}
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "pending_deployments": [
      {
        "algo_package_id": "pkg-fall-detection",
        "algo_name": "fall_detection",
        "version": "1.0.0",
        "download_url": "http://minio:9000/aivision-algorithms/fall-detection-1.0.0.tar.gz?X-Amz-Algorithm=...",
        "expected_md5": "a1b2c3d4e5f6...",
        "extract_path": "/opt/aivision/algorithms/fall_detection/1.0.0"
      }
    ]
  }
}
```

> `pending_deployments` 数组仅在有待下发算法时返回。`download_url` 为 MinIO 预签名 URL，有效期 1 小时。

---

### 触发算法包下发

```
POST /api/v1/edge-nodes/:id/deploy-algo
```

**请求体：**

```json
{
  "algo_package_id": "pkg-fall-detection",
  "version": "1.0.0",
  "algo_name": "fall_detection",
  "download_url": "http://minio:9000/aivision-algorithms/fall-detection-1.0.0.tar.gz",
  "expected_md5": "a1b2c3d4e5f6..."
}
```

| 字段            | 类型   | 必填 | 说明                       |
|-----------------|--------|------|----------------------------|
| algo_package_id | string | 是   | 算法包 ID                  |
| version         | string | 是   | 版本号                     |
| algo_name       | string | 是   | 算法名称                   |
| download_url    | string | 是   | 算法包下载 URL             |
| expected_md5    | string | 是   | 算法包 MD5，用于完整性校验 |

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "pending",
    "message": "算法包已加入下发队列"
  }
}
```

---

### 查询已安装算法

```
GET /api/v1/edge-nodes/:id/algorithms
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "algo_package_id": "pkg-face-recognition",
      "algo_name": "face_recognition",
      "version": "1.0.0",
      "status": "installed",
      "install_path": "/opt/aivision/algorithms/face_recognition/1.0.0",
      "installed_at": "2026-06-14T09:00:00+08:00"
    }
  ]
}
```

状态枚举：

| 状态        | 说明                     |
|-------------|--------------------------|
| pending     | 等待下发                 |
| downloading | 引擎正在下载             |
| installed   | 已安装                   |
| failed      | 安装失败（可自动重试）   |

---

### 推荐节点

```
GET /api/v1/edge-nodes/recommend-node?algo_package_id=<id>
```

根据算法包 ID 查询已安装该算法且当前在线、负载最低的节点。

**查询参数：**

| 参数            | 类型   | 必填 | 说明       |
|-----------------|--------|------|------------|
| algo_package_id | string | 是   | 算法包 ID  |

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "recommended_node_id": "uuid-string",
    "node_name": "edge-node-01",
    "current_load": 1,
    "max_load": 4,
    "load_rate": 0.25
  }
}
```

---

## WebSocket 事件

```
ws://host:8090/ws
```

鉴权通过查询参数传递：`ws://host:8090/ws?token=<jwt-token>`

### 事件类型

| 事件类型              | 数据说明                     |
|-----------------------|------------------------------|
| `edge-node-status`    | 节点状态变更（online/offline） |
| `task-status`         | 任务状态变更（suspended/running） |

### 示例消息

```json
{
  "type": "edge-node-status",
  "timestamp": "2026-06-14T10:00:05+08:00",
  "payload": {
    "node_id": "uuid-string",
    "node_name": "edge-node-01",
    "status": "offline",
    "previous_status": "online"
  }
}
```

```json
{
  "type": "task-status",
  "timestamp": "2026-06-14T10:00:06+08:00",
  "payload": {
    "task_id": "uuid-string",
    "task_name": "车间安全检测任务",
    "status": "suspended",
    "error_reason": "节点 edge-node-01 离线",
    "node_name": "edge-node-01"
  }
}
```
