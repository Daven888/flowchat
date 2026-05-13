# FlowChat：基于 Go 的 AI 多轮对话服务平台

## 项目简介

FlowChat 是一个 Go 后端 + React 前端的 AI 多轮对话服务平台。

核心能力包括：

- 用户注册、登录与 JWT 鉴权
- 用户级 Provider API Key 管理（AES-256-GCM 加密存储）
- 多 Provider / 多模型配置
- 会话管理（创建时校验模型，绑定 user_id）
- 消息生命周期管理（user completed / assistant generating → completed/failed/cancelled）
- 消息写入事务（user + assistant 原子写入）
- SSE 流式对话主链路（meta / message / done / error 事件）
- Redis 同会话生成锁
- Provider 抽象（Mock + OpenAI-compatible）
- 消息分页（before_id + limit）+ 会话内搜索
- 长会话上下文压缩（大模型摘要 + 最近消息）
- Redis Stream 异步旁路任务（call log / usage stat / 自动标题）
- 模型调用日志 + 用户调用统计
- 敏感词过滤
- Markdown 导出
- React 前端演示页面

前端只调用 FlowChat 后端接口，不直接接触任何模型 API Key。

---

## 技术栈

### 后端

| 组件 | 用途 |
|------|------|
| Go 1.25 | 开发语言 |
| Gin | HTTP 框架 |
| GORM | ORM（MySQL） |
| MySQL 8.0 | 数据持久化 |
| Redis 7 | 会话锁 + Stream 异步任务 |
| JWT | 用户认证 |
| bcrypt | 密码哈希 |
| AES-256-GCM | API Key 加密存储 |
| Viper | 配置管理 |
| Zap | 结构化日志 |
| Docker Compose | 本地基础设施 |

### 前端

| 组件 | 用途 |
|------|------|
| React 18 | UI 框架 |
| TypeScript | 类型安全 |
| Vite | 构建工具 |
| Tailwind CSS | 样式 |

---

## 项目目录结构

```
flowchat/
│
├── cmd/server/                   # 后端服务入口
│   └── main.go
│
├── configs/
│   └── config.yaml               # 配置文件
│
├── internal/
│   ├── auth/                     # JWT 生成与解析
│   ├── config/                   # Viper 配置加载
│   ├── event/                    # Redis Stream 事件（发布/消费/DLQ）
│   │   ├── model_call_event.go
│   │   ├── model_call_handler.go
│   │   └── redis_stream.go
│   ├── handler/                  # HTTP 处理器
│   ├── lock/                     # Redis 会话生成锁
│   ├── middleware/               # JWT 鉴权中间件
│   ├── model/                    # GORM 数据模型
│   ├── provider/                 # 模型 Provider
│   │   ├── provider.go           # ChatProvider 接口
│   │   ├── mock/                 # Mock Provider
│   │   └── openai/               # OpenAI Compatible Provider
│   ├── repository/               # 数据库访问层
│   ├── router/                   # 路由注册
│   ├── sensitive/                # 敏感词过滤
│   ├── server/                   # Gin 服务初始化 + CORS
│   └── service/                  # 业务逻辑层
│
├── pkg/
│   ├── cryptoutil/               # AES-256-GCM 加解密
│   ├── logger/                   # Zap 日志初始化
│   ├── mysql/                    # GORM MySQL 连接 + 迁移
│   ├── redis/                    # go-redis 连接
│   └── uuid/                     # UUID 生成
│
├── scripts/
│   ├── init.sql                  # 数据库初始化 DDL
│   ├── test_message_pagination.sh
│   ├── test_compression.sh
│   └── test_transaction_and_events.sh
│
├── docs/
│   ├── api.md                    # API 文档
│   └── backend-features.md      # 后端功能详解
│
├── frontend/                     # React 前端
├── docker-compose.yml            # MySQL + Redis
├── go.mod / go.sum
└── README.md
```

---

## 本地启动方式

### 前置条件

- Go 1.25+
- Node.js 18+
- Docker 和 Docker Compose

### 1. 启动基础设施

```bash
docker-compose up -d
```

启动 MySQL 8.0（端口 3308）和 Redis 7（端口 6380）。

### 2. 设置环境变量

```bash
export FLOWCHAT_CREDENTIAL_SECRET="your-32-byte-secret-key-at-least"
```

`FLOWCHAT_CREDENTIAL_SECRET` 用于 AES-256-GCM 加密用户 API Key，服务启动时必填。密钥通过 SHA-256 派生为 32 字节加密密钥。

### 3. 启动后端

```bash
go run ./cmd/server
```

后端默认运行在 `http://localhost:8080`。

### 4. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认运行在 `http://localhost:5173`。

### 验证后端

```bash
curl http://localhost:8080/ping
# {"message":"pong"}
```

### 配置用户 API Key

登录后，通过 API 配置各 Provider 的 API Key：

```bash
# 为 deepseek provider 配置 API Key
curl -X PUT http://localhost:8080/api/v1/user/credentials/deepseek \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"api_key": "sk-your-deepseek-key"}'
```

Key 在后端以 AES-256-GCM 加密存储，前端只看到 `configured`、`key_suffix`、`status` 等安全信息。

---

## 配置说明

配置文件位于 `configs/config.yaml`。

### MySQL / Redis / JWT

```yaml
mysql:
  host: 127.0.0.1
  port: 3308
  user: root
  password: root
  database: flowchat

redis:
  addr: 127.0.0.1:6380
  password: ""
  db: 0

jwt:
  secret: flowchat-secret
  expire_hours: 24
```

### 聊天配置

```yaml
chat:
  session_lock_ttl_seconds: 180   # 会话生成锁超时
  max_message_length: 4000        # 单条消息最大长度
```

### 凭据加密

```yaml
credential:
  encryption_key_env: FLOWCHAT_CREDENTIAL_SECRET  # 加密密钥环境变量名
```

### Provider 配置

Provider 只包含连接信息，不包含 API Key：

```yaml
ai:
  providers:
    deepseek:
      type: openai_compatible
      base_url: https://api.deepseek.com
      display_name: DeepSeek
    siliconflow:
      type: openai_compatible
      base_url: https://api.siliconflow.cn/v1
      display_name: SiliconFlow
```

字段说明：
- `type`：Provider 类型，目前支持 `openai_compatible`
- `base_url`：模型服务地址
- `display_name`：前端展示名称

### 模型配置

每个模型可独立配置上下文压缩：

```yaml
models:
  - name: mock
    provider: mock
    api_model: mock
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 0
    context_compression:
      enabled: true
      max_messages_before_compress: 30

  - name: deepseek-chat
    provider: deepseek
    api_model: deepseek-chat
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 1
    context_compression:
      enabled: true
      max_messages_before_compress: 30
```

字段说明：
- `name`：对外暴露的模型名称（用户创建会话时传入）
- `provider`：使用的 Provider 配置名
- `api_model`：请求上游 API 时传递的真实模型名
- `enabled`：是否启用，禁用后无法创建新会话
- `max_context_messages`：上下文保留的最近消息数
- `timeout_seconds`：模型调用超时时间
- `max_retries`：Provider 初始调用失败重试次数
- `context_compression.enabled`：是否启用长会话上下文压缩
- `context_compression.max_messages_before_compress`：completed 消息数超过此值时触发压缩

### 敏感词

```yaml
sensitive_words:
  - sensitive_word_1
  - sensitive_word_2
```

---

## 核心流程

### 流式对话主流程

```text
POST /api/v1/chat/sessions/{session_id}/messages/stream
  → JWT 鉴权
  → 校验会话归属当前用户且未删除
  → 校验 model_name 存在且 enabled
  → 敏感词过滤
  → 获取用户该 Provider 的 API Key（mock 跳过）
  → 获取 Redis 同会话生成锁（SETNX）
  → 事务写入 user 消息（completed）+ assistant 占位（generating）
  → 上下文组装（压缩或最近 N 条）
  → 调用 Provider.StreamChat
  → SSE meta 事件（request_id + assistant_message_id）
  → SSE message 事件（逐段文本）
  → 内存聚合完整 assistant 回复
  → 更新 assistant 消息为 completed
  → SSE done 事件
  → 释放 Redis 锁
  → defer 发布 ModelCallFinishedEvent 到 Redis Stream
```

### 异步旁路任务

```text
ChatHandler.Stream defer
  → 发布 ModelCallFinishedEvent 到 Redis Stream (flowchat:model_call_events)
  → Consumer (flowchat-workers) XREADGROUP 消费
  → 写入 model_call_logs
  → 更新 user_model_usage_stats
  → 成功时自动生成会话标题
  → 失败重试（max 3 次）→ 超过后进入 DLQ (flowchat:model_call_events:dlq)
```

---

## 数据库核心表

| 表名 | 用途 |
|------|------|
| `users` | 用户（id, username, email, password_hash, status） |
| `chat_sessions` | 会话（user_id, title, model_name, status） |
| `chat_messages` | 消息（session_id, user_id, role, content, status, token_count） |
| `user_provider_credentials` | 用户 API Key（user_id, provider_name, encrypted_api_key, key_suffix, status） |
| `chat_session_summaries` | 上下文压缩摘要（session_id UNIQUE, content, last_message_id） |
| `model_call_logs` | 模型调用日志（request_id UNIQUE, tokens, latency_ms, status） |
| `user_model_usage_stats` | 用户调用统计（user_id + model_name + stat_date UNIQUE） |

完整 DDL 见 `scripts/init.sql`。

---

## Provider 说明

### Mock Provider

- 不调用外部模型 API，本地生成模拟回复
- 不需要 API Key
- 支持返回摘要（检测首条消息为 system role）
- 适合本地开发和功能测试

### OpenAI Compatible Provider

- 兼容 OpenAI Chat Completions 协议的模型服务
- API Key 由用户配置，不存储在配置文件或 Provider 实例中
- 每次请求从 `ChatRequest.APIKey` 取用，不持有长期引用
- 支持通过配置文件接入多个同类 Provider

### 安全说明

- 真实模型 API Key 由用户通过 `/api/v1/user/credentials` 配置
- 后端使用 `FLOWCHAT_CREDENTIAL_SECRET` 做 AES-256-GCM 加密存储
- `GET /api/v1/user/credentials` 只返回 `provider_name`、`configured`、`key_suffix`、`status`，不返回明文或密文 Key
- `GET /api/v1/models` 不返回 base_url 和任何 API Key
- 调用日志和统计中不记录 API Key
- `ModelCallFinishedEvent` 不包含 API Key、加密凭据和完整 assistant 回复

---

## 测试脚本

| 脚本 | 用途 |
|------|------|
| `scripts/test_message_pagination.sh` | 分页路由分发、参数校验、search q 校验、has_more/next_before_id |
| `scripts/test_compression.sh` | 长会话消息发送、压缩触发、消息列表/搜索/导出不受影响 |
| `scripts/test_transaction_and_events.sh` | 事务原子写入、Redis Stream、consumer group、DLQ、SSE 不受影响 |

使用方式：

```bash
TOKEN="<jwt>" ./scripts/test_message_pagination.sh
TOKEN="<jwt>" ./scripts/test_compression.sh
TOKEN="<jwt>" ./scripts/test_transaction_and_events.sh
```

---

## 注意事项

- **不要提交真实 API Key**：API Key 由用户通过 API 配置，以密文存储在数据库中。
- **环境变量必填**：`FLOWCHAT_CREDENTIAL_SECRET` 服务启动时必须设置，否则 Fatal 退出。
- **清理测试数据**：`docker-compose down -v` 删除所有容器和数据卷，下次启动重建干净数据库。
- **端口冲突**：如 3308 或 6380 已占用，修改 `docker-compose.yml` 端口映射并同步 `configs/config.yaml`。

---

## License

MIT
