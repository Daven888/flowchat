# FlowChat API 文档

Base URL: `http://localhost:8080/api/v1`

所有需要鉴权的接口须携带 Header: `Authorization: Bearer <jwt-token>`

---

## 1. 健康检查

### GET /ping

无需鉴权。

**Response 200:**
```json
{"message": "pong"}
```

---

## 2. 模型列表

### GET /api/v1/models

无需鉴权。返回所有 `enabled: true` 的模型信息，不返回 base_url。

**Response 200:**
```json
{
  "models": [
    {
      "name": "mock",
      "provider": "mock",
      "api_model": "mock",
      "display_name": "Mock",
      "max_context_messages": 10
    }
  ]
}
```

---

## 3. 用户认证

### POST /api/v1/auth/register

**Request:**
```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "123456"
}
```

**Response 200:**
```json
{
  "user": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "status": 1,
    "created_at": "2026-05-13T10:00:00Z"
  },
  "token": "eyJhbGciOi..."
}
```

### POST /api/v1/auth/login

**Request:**
```json
{
  "email": "alice@example.com",
  "password": "123456"
}
```

**Response 200:** 同上，返回 user + token。

---

## 4. 用户信息

### GET /api/v1/user/profile

**Response 200:**
```json
{
  "id": 1,
  "username": "alice",
  "email": "alice@example.com",
  "status": 1,
  "created_at": "2026-05-13T10:00:00Z"
}
```

---

## 5. Provider 凭据管理

所有 `/api/v1/user/credentials` 接口均需 JWT 鉴权。API Key 在后端以 AES-256-GCM 加密存储，前端只能看到安全信息。

### GET /api/v1/user/credentials

列出所有 Provider 的凭据状态。

**Response 200:**
```json
{
  "credentials": [
    {
      "provider_name": "deepseek",
      "configured": true,
      "key_suffix": "abcd",
      "status": 1,
      "created_at": "2026-05-13T10:00:00Z",
      "updated_at": "2026-05-13T10:00:00Z"
    },
    {
      "provider_name": "siliconflow",
      "configured": false,
      "key_suffix": "",
      "status": 0,
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z"
    }
  ]
}
```

注意：响应中 **不返回** `encrypted_api_key` 或任何明文/密文 Key。

### PUT /api/v1/user/credentials/:provider

配置指定 Provider 的 API Key。`:provider` 必须是 `config.yaml` 中配置的 Provider 名称（mock 除外）。

**Request:**
```json
{
  "api_key": "sk-your-deepseek-api-key"
}
```

**Response 200:**
```json
{
  "message": "credential saved",
  "provider_name": "deepseek",
  "key_suffix": "xy12"
}
```

**Error 400:**
- `{"error": "mock provider does not require api key"}` — 不允许为 mock 配置 Key
- `{"error": "invalid provider"}` — Provider 名称不存在

### DELETE /api/v1/user/credentials/:provider

软删除指定 Provider 的凭据（status 设为 0）。

**Response 200:**
```json
{"message": "credential deleted"}
```

---

## 6. 会话管理

所有 `/api/v1/chat/sessions` 接口均需 JWT 鉴权。

### POST /api/v1/chat/sessions

创建新会话。创建时校验 model_name 存在且 enabled。

**Request:**
```json
{
  "title": "我的对话",
  "model_name": "deepseek-chat"
}
```

`title` 可选，默认值为 "新的对话"。

**Response 200:**
```json
{
  "id": 1,
  "user_id": 1,
  "title": "我的对话",
  "model_name": "deepseek-chat",
  "status": 1,
  "created_at": "2026-05-13T10:00:00Z",
  "updated_at": "2026-05-13T10:00:00Z"
}
```

**Error 400:**
- `{"error": "model not found: xxx"}` — model_name 不存在
- `{"error": "model is disabled: xxx"}` — model 已禁用

### GET /api/v1/chat/sessions

查询当前用户的会话列表（仅 status=1），按 updated_at 倒序。

**Response 200:**
```json
{
  "sessions": [
    {
      "id": 1,
      "user_id": 1,
      "title": "我的对话",
      "model_name": "deepseek-chat",
      "status": 1,
      "created_at": "2026-05-13T10:00:00Z",
      "updated_at": "2026-05-13T10:00:00Z"
    }
  ]
}
```

### GET /api/v1/chat/sessions/:session_id

查询单个会话详情。

**Response 200:** 同上单个 session 对象。

**Error 404:** `{"error": "session not found"}` — 不存在、不属于当前用户或已删除。

### DELETE /api/v1/chat/sessions/:session_id

软删除会话（status 设为 0）。

**Response 200:** `{"message": "session deleted"}`

**Error 404:** `{"error": "session not found"}`

---

## 7. 消息

所有消息接口均需 JWT 鉴权，且 session 必须属于当前用户。

### GET /api/v1/chat/sessions/:session_id/messages

游标分页查询消息，按 id ASC（从旧到新）排序。

**Query Parameters:**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `before_id` | int64 | 0 | 返回此 ID 之前的消息；0 或不传返回最新消息 |
| `limit` | int | 50 | 每页条数，最大 100，超过静默截断 |

**Response 200:**
```json
{
  "messages": [
    {
      "id": 100,
      "session_id": 1,
      "user_id": 1,
      "role": "user",
      "content": "你好",
      "status": "completed",
      "token_count": 0,
      "created_at": "2026-05-13T10:00:00Z",
      "updated_at": "2026-05-13T10:00:00Z"
    },
    {
      "id": 101,
      "session_id": 1,
      "user_id": 1,
      "role": "assistant",
      "content": "你好！有什么可以帮助你的？",
      "status": "completed",
      "token_count": 0,
      "created_at": "2026-05-13T10:00:01Z",
      "updated_at": "2026-05-13T10:00:03Z"
    }
  ],
  "has_more": true,
  "next_before_id": 100
}
```

- `has_more`：是否还有更早的消息（true 时可继续翻页）
- `next_before_id`：当前页最旧消息的 ID，作为下一页的 `before_id`
- 空结果时 `next_before_id` 为 0

**Error 400:**
- `{"error": "invalid session id"}`
- `{"error": "invalid before_id"}` — before_id 为负数或非整数
- `{"error": "invalid limit"}` — limit <= 0

### GET /api/v1/chat/sessions/:session_id/messages/search

在指定会话内搜索消息，使用 MySQL LIKE '%keyword%' 匹配 content。

**Query Parameters:**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `q` | string | (必填) | 搜索关键词，trim 后不能为空 |
| `limit` | int | 50 | 每页条数，最大 100 |

**Response 200:**
```json
{
  "messages": [
    {
      "id": 100,
      "session_id": 1,
      "user_id": 1,
      "role": "user",
      "content": "你好",
      "status": "completed",
      "token_count": 0,
      "created_at": "2026-05-13T10:00:00Z",
      "updated_at": "2026-05-13T10:00:00Z"
    }
  ]
}
```

**Error 400:**
- `{"error": "search query cannot be empty"}` — q 为空或仅空格

---

## 8. 流式对话

### POST /api/v1/chat/sessions/:session_id/messages/stream

发送用户消息并获取 SSE 流式回复。

**Request:**
```json
{
  "content": "你好，帮我写一段 Go 代码"
}
```

**Response (SSE, Content-Type: text/event-stream):**

```
event: meta
data: {"request_id":"req_abc12345","assistant_message_id":102}

event: message
data: {"content":"好的"}

event: message
data: {"content":"，这是"}

event: message
data: {"content":"一段 Go 代码..."}

event: done
data: {"message":"completed"}
```

**SSE 事件类型:**

| event | 说明 |
|-------|------|
| `meta` | 流开始，含 request_id 和 assistant_message_id |
| `message` | 增量文本片段 |
| `done` | 正常完成 |
| `error` | 出错（含 error 描述） |

**Error（非 SSE，HTTP 级别）:**
- 400：`{"error": "content cannot be empty"}` / `{"error_code": "SENSITIVE_CONTENT", "message": "..."}`
- 409：`{"error": "当前会话正在生成回复，请稍后再试"}` — 同会话有正在进行的生成

---

## 9. Markdown 导出

### GET /api/v1/chat/sessions/:session_id/export/markdown

导出会话所有 completed 消息为 Markdown 文件。System 角色消息会被跳过。上下文压缩的摘要不会出现在导出中（摘要存储在独立表中）。

**Response 200:**
```
Content-Type: text/markdown; charset=utf-8
Content-Disposition: attachment; filename="flowchat-session-1.md"

# 我的对话

## User

你好

## Assistant

你好！有什么可以帮助你的？
```

---

## 10. 模型调用日志

### GET /api/v1/model-call-logs

分页查询当前用户的模型调用日志。

**Query Parameters:**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page` | int | 1 | 页码 |
| `page_size` | int | 20 | 每页条数，最大 100 |
| `status` | string | "" | 按状态筛选（success/failed/timeout/cancelled） |
| `model_name` | string | "" | 按模型名筛选 |

**Response 200:**
```json
{
  "logs": [
    {
      "id": 1,
      "request_id": "req_abc12345",
      "user_id": 1,
      "session_id": 1,
      "provider": "deepseek",
      "model_name": "deepseek-chat",
      "status": "success",
      "prompt_tokens": 150,
      "completion_tokens": 300,
      "latency_ms": 2500,
      "finish_reason": "stop",
      "started_at": "2026-05-13T10:00:02Z",
      "finished_at": "2026-05-13T10:00:04Z",
      "created_at": "2026-05-13T10:00:04Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### GET /api/v1/model-call-logs/:id

查询单条调用日志（需属于当前用户）。

---

## 11. 用户用量统计

### GET /api/v1/user/usage-stats

查询当前用户的模型用量统计。

**Query Parameters:**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `start_date` | string | "" | 起始日期，格式 2006-01-02 |
| `end_date` | string | "" | 结束日期，格式 2006-01-02 |
| `model_name` | string | "" | 按模型名筛选 |

**Response 200:**
```json
{
  "stats": [
    {
      "stat_date": "2026-05-13",
      "model_name": "deepseek-chat",
      "total_calls": 10,
      "success_calls": 9,
      "failed_calls": 1,
      "timeout_calls": 0,
      "cancelled_calls": 0,
      "prompt_tokens": 1500,
      "completion_tokens": 3000
    }
  ]
}
```

---

## 错误响应格式

所有错误响应遵循统一格式：

```json
{
  "error": "human-readable error message"
}
```

部分场景包含额外字段：

```json
{
  "error_code": "SENSITIVE_CONTENT",
  "message": "输入内容包含不支持的内容"
}
```

常见 HTTP 状态码：

| 状态码 | 场景 |
|--------|------|
| 200 | 成功 |
| 400 | 参数校验失败 |
| 401 | JWT 无效或过期 |
| 404 | 资源不存在或不属于当前用户 |
| 409 | 同会话已有生成在进行 |
| 500 | 内部错误 |
