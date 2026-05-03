# FlowChat：基于 Go 的 AI 多轮对话服务平台

## 项目简介

FlowChat 是一个 Go 后端 + React 前端的 AI 多轮对话服务平台。

项目实现了 AI 对话服务后端中最常见的工程链路：用户发起对话请求，系统保存上下文，调用模型生成回复，流式返回结果，并完整记录生成过程。

核心能力包括：

- 用户注册、登录与 JWT 鉴权
- 会话创建、查询、详情、软删除
- 多轮对话消息管理
- POST + fetch streaming 流式响应
- Mock Provider 和 OpenAI Compatible Provider
- 模型白名单管理
- Redis 同会话生成锁
- 模型调用日志
- 用户调用统计
- 简单敏感词过滤
- 会话标题自动生成
- Markdown 导出
- React 前端演示页面

前端只调用 FlowChat 后端接口，不直接接触任何模型 API Key。

---

## 功能列表

### 用户认证
- 注册（用户名 + 邮箱 + 密码）
- 登录（邮箱 + 密码）
- JWT Token 签发与鉴权中间件
- 当前用户信息查询

### 模型列表
- `GET /api/v1/models` 返回已启用模型
- 配置文件中通过 `enabled` 字段控制是否启用
- 不返回 base_url、api_key_env 和 API Key

### 会话管理
- 创建会话（可选择模型）
- 查询当前用户的会话列表（仅返回未删除的会话，按更新时间倒序）
- 查询会话详情
- 软删除会话（仅标记 status=0，不物理删除）

### 消息管理
- 查询会话历史消息（按 ID 正序）
- 保存 user 消息（status=completed）
- 创建 assistant 占位消息（status=generating）
- 更新 assistant 消息（完成/失败/取消）

### 流式对话
- POST + fetch streaming 实现 SSE 流式响应
- 先返回 meta 事件（request_id + assistant_message_id）
- 持续返回 message 事件（逐段文本）
- 正常完成返回 done 事件
- 出错返回 error 事件

### Provider 接入
- Mock Provider：本地模拟流式输出，不依赖外部 API
- OpenAI Compatible Provider：接入兼容 OpenAI Chat Completions 协议的模型服务
- Provider 抽象接口，支持后续扩展其他模型服务

### Redis 同会话生成锁
- 同一 session 同一时间只允许一个生成请求
- 使用 SETNX 加锁，Lua 脚本原子释放
- 避免多个请求同时读取上下文导致消息顺序错乱

### 模型调用日志
- 每次模型调用写入一条完整日志
- 记录 request_id、user_id、session_id、provider、model_name、status、tokens、latency_ms、error_message、finish_reason 等
- 支持分页查询和按 status/model_name 筛选

### 用户调用统计
- 按 user_id + model_name + stat_date 聚合
- 使用 INSERT ... ON DUPLICATE KEY UPDATE 保证并发安全
- 记录 total_calls、success_calls、failed_calls、timeout_calls、cancelled_calls、tokens
- 支持按日期范围和模型名查询

### 敏感词过滤
- 在调用模型前检查用户输入
- 命中敏感词时返回 error_code=SENSITIVE_CONTENT，不保存消息、不调用 Provider
- 大小写不敏感，简单子串匹配

### 会话标题自动生成
- 首次成功对话后，根据用户第一条消息自动更新会话标题
- 中文按 rune 截断前 20 字，长内容追加 "..."
- 如果用户创建会话时传入了自定义标题，不会覆盖

### Markdown 导出
- 导出当前会话下所有 completed 状态消息
- 按 role 区分 User 和 Assistant 区块
- 文件下载，Content-Type: text/markdown

### 前端页面
- React + TypeScript + Vite + Tailwind CSS
- 注册 / 登录页面
- 侧边栏会话列表 + 创建 + 模型选择
- 聊天窗口 + 流式消息显示
- 调用日志面板 + 用户统计面板
- Markdown 导出按钮

---

## 技术栈

### 后端
| 组件 | 用途 |
|------|------|
| Go | 开发语言 |
| Gin | HTTP 框架 |
| GORM | ORM（MySQL） |
| MySQL | 数据持久化 |
| Redis | 同会话生成锁 |
| JWT | 用户认证 |
| bcrypt | 密码哈希 |
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
├── configs/                      # 配置文件
│   └── config.yaml
│
├── internal/
│   ├── auth/                     # JWT 生成与解析
│   ├── config/                   # Viper 配置加载
│   ├── handler/                  # HTTP 处理器（auth/session/message/chat/log/stat/export/model）
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
│   ├── server/                   # Gin 服务初始化和 CORS
│   └── service/                  # 业务逻辑层
│
├── pkg/
│   ├── logger/                   # Zap 日志初始化
│   ├── mysql/                    # GORM MySQL 连接
│   ├── redis/                    # go-redis 连接
│   └── uuid/                     # UUID v4 生成
│
├── scripts/
│   └── init.sql                  # 数据库初始化
│
├── frontend/                     # React 前端
│   ├── src/
│   │   ├── api/client.ts         # API 客户端（自动 Bearer token）
│   │   ├── context/AuthContext.tsx
│   │   ├── hooks/useSSE.ts       # SSE 流式解析 Hook
│   │   └── components/           # React 组件
│   └── package.json
│
├── docker-compose.yml            # MySQL + Redis
├── go.mod / go.sum
└── README.md
```

---

## 核心流程说明

流式对话主流程：

```text
POST /api/v1/chat/sessions/{session_id}/messages/stream
  -> JWT 鉴权
  -> 校验会话归属当前用户且未删除
  -> 校验 model_name 在白名单中且已启用
  -> 敏感词过滤
  -> 获取 Redis 同会话生成锁（SETNX）
  -> 在事务中保存 user 消息（completed）+ 创建 assistant 占位消息（generating）
  -> 读取最近 N 条 completed 消息作为上下文
  -> 根据 model_name 选择 Provider
  -> 创建 context.WithTimeout
  -> 调用 Provider.StreamChat
  -> SSE 返回 meta 事件（request_id + assistant_message_id）
  -> SSE 持续返回 message 事件
  -> 内存中聚合完整 assistant 回复
  -> 更新 assistant 消息为 completed
  -> 写入 model_call_logs
  -> 更新 user_model_usage_stats
  -> 自动生成会话标题（如果是首次对话）
  -> 释放 Redis 会话生成锁
```

---

## Provider 说明

### Mock Provider
- 不调用外部模型 API
- 在本地生成模拟回复，每隔 100ms 输出一小段文本
- 不需要任何 API Key
- 适合本地开发、流式功能测试和演示

### OpenAI Compatible Provider
- 用于接入所有兼容 OpenAI Chat Completions 协议的模型服务
- 通过配置文件 `ai.providers` 指定 base_url 和 api_key_env
- API Key 从后端环境变量读取，不写入配置文件
- 支持多个 provider 实例（如 openai_compatible、compatible_example）

### 安全说明
- 真实模型 API Key 只通过后端环境变量读取
- 配置文件中的 `api_key_env` 只是环境变量名称
- 前端不保存、不传递、不展示真实模型 API Key
- `GET /api/v1/models` 不返回 base_url、api_key_env 和任何 API Key
- 调用日志和统计中不记录 API Key

---

## 本地启动方式

### 前置条件
- Go 1.21+
- Node.js 18+
- Docker 和 Docker Compose
- MySQL 和 Redis 本地端口未被占用（或修改 docker-compose.yml 中的端口映射）

### 1. 启动基础设施

```bash
docker-compose up -d
```

这会启动 MySQL 8.0（端口 3308）和 Redis 7（端口 6380）。

### 2. 启动后端

```bash
go run ./cmd/server
```

后端默认运行在 `http://localhost:8080`。

如果需要使用真实模型，先设置环境变量：

```powershell
# Windows PowerShell
$env:DEEPSEEK_API_KEY = "your-api-key"
```

```bash
# Linux / macOS
export DEEPSEEK_API_KEY="your-api-key"
```

然后在 `configs/config.yaml` 中将对应模型的 `enabled` 改为 `true`。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认运行在 `http://localhost:5173`。

### 访问地址

| 服务 | 地址 |
|------|------|
| 后端 API | http://localhost:8080 |
| 前端页面 | http://localhost:5173 |

### 验证后端

```bash
curl http://localhost:8080/ping
# {"message":"pong"}
```

---

## 配置说明

配置文件位于 `configs/config.yaml`。

### MySQL 配置

```yaml
mysql:
  host: 127.0.0.1
  port: 3308
  user: root
  password: root
  database: flowchat
```

### Redis 配置

```yaml
redis:
  addr: 127.0.0.1:6380
  password: ""
  db: 0
```

### JWT 配置

```yaml
jwt:
  secret: flowchat-secret
  expire_hours: 24
```

### 会话配置

```yaml
chat:
  session_lock_ttl_seconds: 180
  max_message_length: 4000
```

### Provider 配置

```yaml
ai:
  providers:
    openai_compatible:
      type: openai_compatible
      base_url: https://api.deepseek.com
      api_key_env: DEEPSEEK_API_KEY
```

字段说明：
- `ai.providers.{name}`：Provider 配置名，供 models 引用
- `type`：Provider 类型，目前支持 `openai_compatible`
- `base_url`：模型服务地址
- `api_key_env`：从哪个环境变量读取 API Key

### 模型白名单

```yaml
models:
  - name: mock
    provider: mock
    api_model: mock
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 0

  - name: deepseek-v4-pro
    provider: openai_compatible
    api_model: deepseek-chat
    enabled: false
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 1
```

字段说明：
- `name`：对外暴露的模型名称（用户创建会话时传入）
- `provider`：使用的 Provider 配置名
- `api_model`：请求上游 API 时传递的真实模型名
- `enabled`：是否启用
- `max_context_messages`：上下文读取条数
- `timeout_seconds`：模型调用超时时间
- `max_retries`：Provider 初始调用失败重试次数

### 敏感词配置

```yaml
sensitive_words:
  - sensitive_word_1
  - sensitive_word_2
```

---

## 使用流程

1. 打开浏览器访问 `http://localhost:5173`
2. 注册新账号（用户名 + 邮箱 + 密码），或使用已有账号登录
3. 左侧面板下拉菜单选择模型（mock 默认启用）
4. 点击 `+` 创建新会话
5. 在聊天框输入消息，按 Enter 发送
6. 观察 assistant 回复逐字流式显示
7. 右侧面板切换 **Call Logs** 查看调用日志，切换 **Usage Stats** 查看使用统计
8. 点击聊天窗口顶部 **Export Markdown** 导出对话为 `.md` 文件


## 注意事项

- **不要提交真实 API Key**：`configs/config.yaml` 中的 `api_key_env` 只是环境变量名称，真实 Key 通过环境变量传入，不会出现在配置文件中。
- **清理测试数据**：可以通过 `docker-compose down -v` 删除所有容器和数据卷，下次 `docker-compose up -d` 会重建干净的数据库。
- **端口冲突**：如果 3308 或 6380 端口已被占用，可以修改 `docker-compose.yml` 中 MySQL 和 Redis 的宿主机端口映射，并同步修改 `configs/config.yaml` 中对应的端口号。

---

## License

MIT
