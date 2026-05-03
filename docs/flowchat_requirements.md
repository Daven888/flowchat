# FlowChat：基于 Go 的 AI 多轮对话服务平台

## 1. 项目定位

**FlowChat** 是一个基于 Go 的 AI 多轮对话服务平台。

用户注册并登录后，可以创建 AI 会话，在会话中连续发送消息，并通过流式响应实时接收模型回复。系统负责维护用户、会话、消息、模型 Provider、生成状态、调用日志、调用统计和对话导出等核心能力。

本项目不做复杂 Agent 编排、RAG 知识库、MCP 工具调用、图片识别、前端页面、真实支付计费和 Kubernetes 部署。项目重点是完成一个 AI Chat 后端的完整核心链路。

本项目包含以下核心能力：

- 用户注册、登录与 JWT 鉴权；
- 会话创建、查询、详情查看和软删除；
- 多轮对话消息管理；
- POST + fetch streaming 流式响应；
- 聊天记录持久化；
- Mock Provider，用于本地无真实 API Key 时完整跑通流程；
- OpenAI 兼容 Provider，用于接入真实模型服务；
- Provider 抽象层，用于屏蔽不同模型服务的调用差异；
- 模型白名单管理，限制用户只能调用后端支持的模型；
- 同会话生成并发控制；
- 模型调用超时控制、客户端取消处理和失败重试；
- 简单敏感词过滤；
- 会话标题自动生成；
- 用户调用次数统计；
- 对话导出为 Markdown；
- 模型调用日志记录。

项目的核心目标不是做一个复杂的大模型应用，而是实现 AI 对话服务后端中最常见、最关键的工程链路：**用户发起对话请求，系统保存上下文，调用模型生成回复，流式返回结果，并完整记录生成过程。**

---

## 2. 核心业务流程

用户完成注册和登录后，可以创建一个 AI 会话。每个会话绑定一个模型名称，例如 `mock`、`gpt-4o-mini` 或其他后端白名单中支持的模型。

用户在会话中发送消息后，系统执行以下流程：

1. 校验 JWT，识别当前用户；
2. 校验会话是否存在、未删除，并且属于当前用户；
3. 校验会话绑定的模型是否在后端模型白名单中；
4. 对用户输入进行基础参数校验和敏感词过滤；
5. 为本次模型调用生成 `request_id`；
6. 获取当前会话的 Redis 生成锁，保证同一个 session 同一时间只有一个生成请求；
7. 在同一个数据库事务中保存 user 消息，并创建 assistant 占位消息；
8. 查询当前会话最近 N 条已完成消息作为上下文；
9. 根据模型名称选择对应 Provider；
10. 使用带超时控制的 context 调用 Provider；
11. 服务端通过 POST + fetch streaming 持续返回模型输出；
12. 在内存中聚合完整 assistant 回复内容；
13. 模型生成完成后，更新 assistant 消息状态为 `completed`；
14. 如果生成失败、超时或客户端断开，则更新 assistant 消息为对应状态；
15. 写入模型调用日志；
16. 更新用户调用次数统计；
17. 如果是会话第一轮对话，则根据用户首条消息自动生成会话标题；
18. 释放当前会话的 Redis 生成锁。

整个流程围绕一条主链路展开：

```text
用户发送消息
  -> 参数校验与敏感词过滤
  -> 保存 user 消息
  -> 创建 assistant 占位消息
  -> 读取历史上下文
  -> 调用模型 Provider
  -> 流式返回内容
  -> 更新 assistant 消息
  -> 记录调用日志和统计信息
```

---

## 3. 功能范围

### 3.1 用户模块

用户模块负责注册、登录和当前用户信息查询。

需要实现的功能包括：

- 用户注册；
- 用户登录；
- JWT Token 签发；
- JWT 鉴权中间件；
- 查询当前用户信息。

用户注册时需要校验用户名和邮箱唯一性。密码不能明文存储，需要使用 bcrypt 等方式进行哈希处理。

用户登录成功后，系统返回 JWT Token。后续访问会话、消息、流式对话、导出等接口时，都需要在请求头中携带 Token。

---

### 3.2 会话模块

会话模块负责管理用户的 AI 对话会话。

需要实现的功能包括：

- 创建会话；
- 查询会话列表；
- 查询会话详情；
- 软删除会话；
- 自动生成会话标题。

会话不进行物理删除。删除会话时，只更新 `chat_sessions.status = 0`。被删除的会话不再出现在用户会话列表中，但历史消息仍保留在数据库中，便于后续审计或排查问题。

会话创建时可以传入 `title` 和 `model_name`。如果用户没有传入标题，则系统使用默认标题，例如“新的对话”。当用户发送第一条消息并完成模型回复后，系统根据用户首条消息自动更新会话标题。

标题生成采用简单规则即可，不依赖额外模型调用。例如：

- 去除用户首条消息前后空格；
- 截取前 20 个中文字符或前 40 个英文字符；
- 如果内容为空，则使用“新的对话”；
- 如果内容过长，则在末尾加省略号。

示例：

```text
用户首条消息：请用 Go 写一个简单的 HTTP 服务
自动标题：请用 Go 写一个简单的 HTTP 服务
```

这个设计可以避免额外模型调用成本，同时保证会话列表具有基本可读性。

---

### 3.3 消息模块

消息模块负责保存用户消息、assistant 消息和消息状态。

消息角色包括：

- `user`：用户发送的消息；
- `assistant`：AI 生成的回复；
- `system`：系统提示词，当前项目保留角色设计，但默认不提供单独接口维护。

消息状态包括：

- `generating`：生成中；
- `completed`：生成完成；
- `failed`：生成失败；
- `cancelled`：客户端断开或请求被取消。

用户发送消息后，系统会先保存一条 `user` 消息，状态为 `completed`。随后系统创建一条 `assistant` 占位消息，状态为 `generating`。

`assistant` 占位消息创建时还没有生成内容，因此 `content` 使用空字符串，不能使用 `NULL`。当模型生成完成后，再将完整回复内容写回 `content` 字段，并将状态更新为 `completed`。

保存 user 消息和创建 assistant 占位消息必须放在同一个数据库事务中，避免出现“用户消息保存成功，但 assistant 占位消息创建失败”的不完整状态。

查询历史消息时，默认返回当前会话下未删除的全部消息，并按消息 ID 正序排列。

---

### 3.4 AI 流式对话模块

AI 流式对话是本项目的核心模块。

核心接口：

```http
POST /api/v1/chat/sessions/{session_id}/messages/stream
```

本项目使用 **POST + fetch streaming** 实现流式响应，不使用浏览器原生 `EventSource`。

原因是浏览器原生 `EventSource` 只支持 GET 请求，不方便在请求体中提交用户消息、模型参数等复杂 JSON 数据。使用 POST 请求可以更自然地提交用户消息，同时服务端仍然按照 SSE 格式持续返回数据。

流式对话请求示例：

```json
{
  "content": "请用 Go 写一个简单的 HTTP 服务"
}
```

流式响应示例：

```text
event: meta
data: {"request_id":"req_xxx","assistant_message_id":1002}

event: message
data: {"content":"可以"}

event: message
data: {"content":"，下面是一个简单示例"}

event: done
data: {"message":"completed"}
```

异常响应示例：

```text
event: error
data: {"error_code":"MODEL_TIMEOUT","message":"模型调用超时"}
```

接口处理逻辑：

1. 校验 JWT；
2. 校验会话属于当前用户；
3. 校验会话未被删除；
4. 校验模型名称在白名单中；
5. 校验用户输入不能为空、不能超过长度限制；
6. 执行敏感词过滤；
7. 获取同会话生成锁；
8. 保存 user 消息并创建 assistant 占位消息；
9. 查询最近 N 条上下文；
10. 根据模型名称选择 Provider；
11. 使用 context.WithTimeout 控制模型调用时间；
12. 调用 Provider.StreamChat；
13. 向客户端发送 meta 事件；
14. 持续读取 Provider 返回的 chunk，并发送 message 事件；
15. 在内存中聚合完整 assistant 内容；
16. 生成完成后更新 assistant 消息状态；
17. 记录模型调用日志；
18. 更新用户调用次数统计；
19. 释放 Redis 生成锁。

---

### 3.5 Provider 模块

Provider 模块用于屏蔽不同模型服务的调用差异。

业务层不直接依赖 OpenAI、DeepSeek 或其他模型厂商的具体 SDK，而是依赖统一的 Provider 接口。

Provider 抽象接口示例：

```go
type ChatProvider interface {
    StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}

type ChatRequest struct {
    RequestID string
    ModelName string
    Messages  []ProviderMessage
}

type ProviderMessage struct {
    Role    string
    Content string
}

type ChatChunk struct {
    Content          string
    Done             bool
    Err              error
    FinishReason     string
    PromptTokens     int
    CompletionTokens int
}
```

Provider 调用错误分为两类：

1. 初始调用失败；
2. 流式生成过程中失败。

如果 Provider 在建立请求时失败，例如配置错误、API Key 缺失、网络连接失败，则通过 `StreamChat` 返回的 `error` 表示。

如果流式生成过程中失败，例如模型服务中断、读取响应失败，则通过 `ChatChunk.Err` 表示。

Provider 必须监听 `ctx.Done()`。当请求超时或客户端断开连接时，Provider 应尽快停止生成，避免后端继续无意义消耗资源。

本项目至少实现两个 Provider：

- Mock Provider；
- OpenAI 兼容 Provider。

---

### 3.6 Mock Provider

Mock Provider 用于本地开发和演示，保证没有真实 API Key 时也能完整跑通系统。

Mock Provider 不调用外部模型服务，而是在本地模拟流式输出。例如每隔 100ms 返回一小段文本，直到完整回复输出完成。

Mock Provider 示例回复：

```text
这是 Mock Provider 生成的回复。你刚才的问题是：xxx
```

Mock Provider 需要支持：

- 正常流式输出；
- 模拟生成延迟；
- 响应 ctx 取消；
- 可选的错误模拟，用于测试失败流程。

Mock Provider 的价值是：

- 不依赖真实模型 API Key；
- 方便本地开发；
- 方便测试流式返回；
- 方便测试超时、取消和失败状态；
- 方便面试演示项目完整链路。

---

### 3.7 OpenAI 兼容 Provider

OpenAI 兼容 Provider 用于接入真实模型服务。

很多模型服务都提供 OpenAI 兼容接口，例如部分云厂商、代理服务或本地模型网关。因此本项目不需要为每个厂商单独写一套逻辑，只需要实现一个 OpenAI 兼容 Provider。

配置示例：

```yaml
ai:
  providers:
    openai_compatible:
      base_url: "https://api.openai.com/v1"
      api_key: "${OPENAI_API_KEY}"
      timeout_seconds: 120
```

OpenAI 兼容 Provider 需要完成：

- 根据配置读取 base_url 和 api_key；
- 将系统内部的 `ProviderMessage` 转换为 OpenAI chat messages；
- 发起流式 chat completions 请求；
- 解析流式响应；
- 将模型返回内容转换为 `ChatChunk`；
- 处理 HTTP 错误、网络错误和读取错误；
- 监听 context 取消；
- 记录 finish_reason 和 token 使用情况。

如果当前环境没有配置真实 API Key，则系统仍然可以使用 Mock Provider 完整运行。

---

### 3.8 模型白名单管理

模型白名单用于限制用户只能选择后端支持的模型，不能随意传入模型名称。

系统可以在配置文件中维护模型列表。

示例：

```yaml
models:
  - name: "mock"
    provider: "mock"
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 0

  - name: "gpt-4o-mini"
    provider: "openai_compatible"
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 1
```

模型白名单字段说明：

- `name`：对外暴露的模型名称；
- `provider`：底层 Provider 类型；
- `enabled`：是否启用；
- `max_context_messages`：上下文读取条数；
- `timeout_seconds`：模型调用超时时间；
- `max_retries`：失败重试次数。

当用户创建会话或调用对话接口时，系统必须校验 `model_name` 是否存在且已启用。

系统提供模型列表接口：

```http
GET /api/v1/models
```

该接口只返回已启用模型，供客户端创建会话时选择。

---

### 3.9 同会话并发控制

本项目限制同一个 session 同一时间只允许一个生成中的请求。

原因是多轮对话依赖上下文顺序。如果同一个会话中多个生成请求同时执行，可能出现以下问题：

- 多个请求同时读取旧上下文；
- assistant 回复顺序错乱；
- 后完成的请求覆盖先完成的状态；
- 历史消息顺序不符合用户真实对话顺序。

因此系统使用 Redis 简单锁控制同会话生成并发。

Redis key 设计：

```text
chat:session:generating:{session_id}
```

请求进入时执行：

```text
SETNX chat:session:generating:{session_id} request_id EX 180
```

如果锁已存在，说明当前会话已经有回复正在生成，接口直接返回错误：

```json
{
  "error": "当前会话正在生成回复，请稍后再试"
}
```

锁 TTL 设置为 180 秒。模型调用超时时间必须小于锁 TTL，避免模型仍在生成时锁提前过期。

释放锁时不能直接删除 key，必须先判断锁中的 value 是否等于当前请求的 `request_id`。只有确认锁仍由当前请求持有时，才能删除锁。实际实现中可以使用 Redis Lua 脚本保证“比较 value + 删除 key”的原子性。

释放锁的 Lua 脚本示例：

```lua
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
```

更复杂的方案可以做锁续期，但本项目通过“锁 TTL 大于模型超时时间”的方式控制复杂度。

---

### 3.10 上下文读取策略

本项目直接从 MySQL 查询最近 N 条 completed 消息作为上下文，不使用 Redis 缓存上下文。

原因是聊天消息本身已经持久化在 MySQL 中，直接查询可以保证逻辑简单、数据可靠，也方便面试时讲清楚。

上下文读取规则：

1. 用户消息先落库，状态为 `completed`；
2. 查询当前会话最近 N 条 `completed` 消息；
3. 查询时排除 `failed`、`cancelled`、`generating` 状态消息；
4. SQL 按 `id DESC LIMIT N` 查询；
5. 代码中反转为正序；
6. 按从旧到新的顺序组装为模型上下文。

示例逻辑：

```text
保存当前 user 消息
  -> 查询最近 10 条 completed 消息
  -> 反转为正序
  -> 组装为 Provider 请求
  -> 调用模型
```

`failed`、`cancelled`、`generating` 状态的消息不进入模型上下文，避免模型基于不完整或错误的内容继续生成。

---

### 3.11 敏感词过滤

敏感词过滤用于在调用模型前进行基础内容校验。

本项目只实现简单敏感词过滤，不做复杂内容安全系统。

处理逻辑：

1. 用户提交消息；
2. 系统检查消息是否为空、是否超过最大长度；
3. 系统检查消息中是否包含敏感词；
4. 如果命中敏感词，直接拒绝请求，不保存消息，不调用模型；
5. 返回统一错误响应。

示例错误响应：

```json
{
  "error_code": "SENSITIVE_CONTENT",
  "message": "输入内容包含不支持的内容"
}
```

敏感词可以先在配置文件中维护：

```yaml
sensitive_words:
  - "敏感词1"
  - "敏感词2"
```

本项目的敏感词过滤重点不是做复杂算法，而是体现 AI 后端在调用模型前需要有基础输入治理能力。

---

### 3.12 模型调用超时、取消与失败重试

每次模型调用都必须设置超时时间。

系统根据模型白名单配置中的 `timeout_seconds` 创建 context：

```go
ctx, cancel := context.WithTimeout(parentCtx, timeout)
defer cancel()
```

如果模型调用超过超时时间，系统需要：

- 停止读取 Provider 输出；
- 向客户端发送 error 事件；
- 将 assistant 消息状态更新为 `failed`；
- 在调用日志中记录 `status = timeout`；
- 释放当前会话生成锁。

如果客户端主动断开连接，Gin 请求上下文会被取消。系统需要监听请求 context：

- 停止 Provider 调用；
- 将 assistant 消息状态更新为 `cancelled`；
- 在调用日志中记录 `status = cancelled`；
- 释放当前会话生成锁。

失败重试只针对 Provider 初始调用失败或临时网络错误，不对已经开始输出内容的流式生成过程进行重试。

原因是流式输出一旦已经返回给客户端，如果中途重试，可能造成内容重复或上下文混乱。

重试规则：

- Provider 初始调用失败，可以按模型配置中的 `max_retries` 重试；
- 流式过程中出现 `ChatChunk.Err`，不重试，直接标记失败；
- 超时不重试；
- 客户端取消不重试；
- Mock Provider 默认不重试。

---

### 3.13 模型调用日志

模型调用日志用于记录每一次模型调用的完整过程，方便问题排查、性能分析和后续统计。

每次调用都需要记录：

- request_id；
- user_id；
- session_id；
- provider；
- model_name；
- status；
- prompt_tokens；
- completion_tokens；
- latency_ms；
- error_code；
- error_message；
- finish_reason；
- started_at；
- finished_at。

日志状态包括：

- `success`：调用成功；
- `failed`：调用失败；
- `timeout`：模型调用超时；
- `cancelled`：客户端断开或请求被取消。

`request_id` 由服务端生成 UUID，主要用于模型调用日志追踪和问题排查，因此可以保证全局唯一。

需要注意的是，服务端生成的 `request_id` 不等同于客户端请求幂等标识。如果后续要支持客户端失败重试幂等，可以在请求体中增加 `client_request_id`，并对 `user_id + session_id + client_request_id` 建立唯一约束。

---

### 3.14 用户调用次数统计

用户调用次数统计用于记录用户维度的模型使用情况。

每次模型调用结束后，无论成功、失败、超时还是取消，都可以更新用户调用统计。这样可以分析用户实际发起了多少次模型请求。

统计粒度按用户、模型和日期聚合。

例如：

```text
user_id + model_name + stat_date
```

统计字段包括：

- total_calls：总调用次数；
- success_calls：成功次数；
- failed_calls：失败次数；
- timeout_calls：超时次数；
- cancelled_calls：取消次数；
- prompt_tokens：输入 token 总数；
- completion_tokens：输出 token 总数。

调用统计可以和模型调用日志配合使用：

- `model_call_logs` 记录每一次调用的明细；
- `user_model_usage_stats` 记录用户维度的聚合结果。

这样既能查单次请求详情，也能查用户整体使用情况。

---

### 3.15 对话导出为 Markdown

系统支持将单个会话导出为 Markdown 文本。

导出接口：

```http
GET /api/v1/chat/sessions/{session_id}/export/markdown
```

导出规则：

1. 校验 JWT；
2. 校验会话属于当前用户；
3. 查询当前会话全部消息；
4. 按消息 ID 正序排列；
5. 只导出 `completed` 状态的消息；
6. 生成 Markdown 文本；
7. 返回文件下载响应。

Markdown 示例：

```markdown
# 请用 Go 写一个简单的 HTTP 服务

## User

请用 Go 写一个简单的 HTTP 服务

## Assistant

下面是一个简单示例：

```go
package main

func main() {
    // ...
}
```

```
导出功能不需要引入复杂文件存储，直接在接口中动态生成 Markdown 内容即可。

---

## 4. 技术选型

本项目技术栈：

- Go；
- Gin；
- GORM；
- MySQL；
- Redis；
- JWT；
- bcrypt；
- Viper；
- Zap；
- Docker Compose。

技术使用原则：

- Gin 负责 HTTP 路由、中间件和流式响应；
- GORM 负责 MySQL 数据访问；
- MySQL 保存用户、会话、消息、调用日志和调用统计；
- Redis 用于同会话生成锁；
- JWT 用于用户认证；
- bcrypt 用于密码哈希；
- Viper 用于读取配置文件；
- Zap 用于结构化日志；
- Docker Compose 用于本地启动 MySQL 和 Redis；
- Provider 抽象层负责屏蔽不同模型服务差异。

---

## 5. 核心表设计

### 5.1 users 用户表

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(128) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    status TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

字段说明：

- `status = 1`：正常；
- `status = 0`：禁用。

---

### 5.2 chat_sessions 会话表

```sql
CREATE TABLE chat_sessions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL,
    model_name VARCHAR(64) NOT NULL,
    status TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_user_updated (user_id, updated_at)
);
```

字段说明：

- `user_id`：会话所属用户；
- `title`：会话标题；
- `model_name`：会话使用的模型名称；
- `status = 1`：正常；
- `status = 0`：已删除。

---

### 5.3 chat_messages 消息表

```sql
CREATE TABLE chat_messages (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    session_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL,
    content MEDIUMTEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'completed',
    error_message VARCHAR(255) NULL,
    token_count INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    INDEX idx_session_id_id (session_id, id),
    INDEX idx_user_created (user_id, created_at)
);
```

字段说明：

- `role`：消息角色，可取 `user`、`assistant`、`system`；
- `content`：消息内容；
- `status`：消息状态，可取 `generating`、`completed`、`failed`、`cancelled`；
- `error_message`：失败原因；
- `token_count`：消息 token 数，可以由 Provider 返回或简单估算。

设计说明：

- `content` 使用 `MEDIUMTEXT`，避免普通 `TEXT` 在长回复场景下不够用；
- assistant 占位消息创建时，`content` 使用空字符串；
- 查询上下文时只读取 `status = 'completed'` 的消息；
- 查询最近 N 条消息时使用 `(session_id, id)` 索引。

---

### 5.4 model_call_logs 模型调用日志表

```sql
CREATE TABLE model_call_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    request_id VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    session_id BIGINT NOT NULL,
    provider VARCHAR(64) NOT NULL,
    model_name VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    prompt_tokens INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    error_code VARCHAR(64) NULL,
    error_message VARCHAR(255) NULL,
    finish_reason VARCHAR(64) NULL,
    started_at DATETIME NOT NULL,
    finished_at DATETIME NULL,
    created_at DATETIME NOT NULL,
    UNIQUE KEY uk_request_id (request_id),
    INDEX idx_user_created (user_id, created_at),
    INDEX idx_session_id (session_id)
);
```

字段说明：

- `request_id`：服务端生成的调用追踪 ID；
- `provider`：底层 Provider 类型，例如 `mock`、`openai_compatible`；
- `model_name`：模型名称；
- `status`：调用状态，可取 `success`、`failed`、`timeout`、`cancelled`；
- `finish_reason`：模型结束原因，例如 `stop`、`length`、`error`。

---

### 5.5 user_model_usage_stats 用户模型调用统计表

```sql
CREATE TABLE user_model_usage_stats (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    model_name VARCHAR(64) NOT NULL,
    stat_date DATE NOT NULL,
    total_calls INT NOT NULL DEFAULT 0,
    success_calls INT NOT NULL DEFAULT 0,
    failed_calls INT NOT NULL DEFAULT 0,
    timeout_calls INT NOT NULL DEFAULT 0,
    cancelled_calls INT NOT NULL DEFAULT 0,
    prompt_tokens BIGINT NOT NULL DEFAULT 0,
    completion_tokens BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE KEY uk_user_model_date (user_id, model_name, stat_date),
    INDEX idx_user_date (user_id, stat_date)
);
```

设计说明：

- 该表用于按天统计用户模型调用情况；
- 每次模型调用完成后，根据调用结果更新对应统计记录；
- 如果当天记录不存在，则先插入再更新；
- 更新时需要注意并发安全，可以使用数据库唯一约束配合 `ON DUPLICATE KEY UPDATE`。

---

## 6. 核心接口设计

### 6.1 用户接口

```http
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/user/profile
```

注册请求：

```json
{
  "username": "test_user",
  "email": "test@example.com",
  "password": "123456"
}
```

登录请求：

```json
{
  "email": "test@example.com",
  "password": "123456"
}
```

---

### 6.2 模型接口

```http
GET /api/v1/models
```

返回示例：

```json
{
  "models": [
    {
      "name": "mock",
      "provider": "mock"
    },
    {
      "name": "gpt-4o-mini",
      "provider": "openai_compatible"
    }
  ]
}
```

该接口只返回配置中已启用的模型。

---

### 6.3 会话接口

```http
POST   /api/v1/chat/sessions
GET    /api/v1/chat/sessions
GET    /api/v1/chat/sessions/{session_id}
DELETE /api/v1/chat/sessions/{session_id}
```

创建会话请求：

```json
{
  "title": "新的对话",
  "model_name": "mock"
}
```

如果 `title` 为空，系统使用默认标题“新的对话”。

`model_name` 必须是后端模型白名单中已启用的模型，不能由用户随意传入。

---

### 6.4 消息接口

```http
GET  /api/v1/chat/sessions/{session_id}/messages
POST /api/v1/chat/sessions/{session_id}/messages/stream
```

查询历史消息返回当前会话下的消息列表，按消息 ID 正序排列。

流式对话请求：

```json
{
  "content": "请用 Go 写一个简单的 HTTP 服务"
}
```

---

### 6.5 调用日志接口

```http
GET /api/v1/model-call-logs
GET /api/v1/model-call-logs/{id}
```

日志查询接口用于查看当前用户的模型调用记录。

可以支持基础查询参数：

```text
status
model_name
start_time
end_time
page
page_size
```

---

### 6.6 用户调用统计接口

```http
GET /api/v1/user/usage-stats
```

查询当前用户的模型调用统计。

可以支持查询参数：

```text
start_date
end_date
model_name
```

返回示例：

```json
{
  "items": [
    {
      "stat_date": "2026-05-03",
      "model_name": "mock",
      "total_calls": 10,
      "success_calls": 9,
      "failed_calls": 1,
      "timeout_calls": 0,
      "cancelled_calls": 0,
      "prompt_tokens": 1200,
      "completion_tokens": 3000
    }
  ]
}
```

---

### 6.7 Markdown 导出接口

```http
GET /api/v1/chat/sessions/{session_id}/export/markdown
```

该接口返回 Markdown 文件内容。

响应头示例：

```http
Content-Type: text/markdown; charset=utf-8
Content-Disposition: attachment; filename="flowchat-session-1001.md"
```

---

## 7. 流式对话链路

完整链路如下：

```text
请求进入
  -> JWT 鉴权
  -> 生成 request_id
  -> 校验 session 属于当前用户且未删除
  -> 校验 model_name 在白名单中
  -> 校验输入内容
  -> 敏感词过滤
  -> 获取 Redis 会话生成锁
  -> 在事务中保存 user 消息并创建 assistant 占位消息
  -> 读取最近 N 条 completed 消息作为上下文
  -> 根据 model_name 选择 Provider
  -> 创建 context.WithTimeout
  -> 调用 Provider.StreamChat
  -> SSE 返回 meta 事件，包含 request_id 和 assistant_message_id
  -> SSE 持续返回 message 事件
  -> 内存中聚合完整 assistant 回复
  -> 生成完成后更新 assistant 消息为 completed
  -> 如果是首轮对话，则自动更新会话标题
  -> 写 model_call_logs
  -> 更新 user_model_usage_stats
  -> 释放 Redis 会话生成锁
```

失败处理：

- 敏感词过滤失败：不保存消息，不调用 Provider；
- 获取会话生成锁失败：直接返回“当前会话正在生成回复”；
- Provider 初始调用失败：根据配置进行有限重试；
- Provider 初始调用最终失败：assistant 消息更新为 `failed`；
- 流式过程中出现 `ChatChunk.Err`：assistant 消息更新为 `failed`；
- 模型调用超时：assistant 消息更新为 `failed`，日志状态为 `timeout`；
- 客户端断开连接：assistant 消息更新为 `cancelled`，日志状态为 `cancelled`；
- 无论成功、失败、超时还是取消，都需要释放 Redis 锁并记录调用日志。

---

## 8. 项目目录结构建议

推荐目录结构：

```text
flowchat/
  cmd/
    server/
      main.go
  configs/
    config.yaml
  internal/
    config/
    server/
    router/
    middleware/
    handler/
    service/
    repository/
    model/
    provider/
      mock/
      openai/
    auth/
    lock/
    sensitive/
    export/
    stats/
    response/
  pkg/
    logger/
    uuid/
  scripts/
    init.sql
  docker-compose.yml
  go.mod
  README.md
```

目录说明：

- `cmd/server`：服务启动入口；
- `configs`：配置文件；
- `internal/router`：路由注册；
- `internal/middleware`：JWT 鉴权、错误处理等中间件；
- `internal/handler`：HTTP 接口层；
- `internal/service`：业务逻辑层；
- `internal/repository`：数据库访问层；
- `internal/model`：数据库实体；
- `internal/provider`：模型 Provider 抽象和实现；
- `internal/lock`：Redis 锁；
- `internal/sensitive`：敏感词过滤；
- `internal/export`：Markdown 导出；
- `internal/stats`：调用统计；
- `pkg/logger`：日志初始化；
- `scripts/init.sql`：数据库初始化 SQL。

---

## 9. 配置文件设计

配置文件示例：

```yaml
server:
  port: 8080

mysql:
  dsn: "root:root@tcp(127.0.0.1:3306)/flowchat?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

jwt:
  secret: "flowchat-secret"
  expire_hours: 24

chat:
  session_lock_ttl_seconds: 180
  max_message_length: 4000

models:
  - name: "mock"
    provider: "mock"
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 0

  - name: "gpt-4o-mini"
    provider: "openai_compatible"
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 1

ai:
  providers:
    openai_compatible:
      base_url: "https://api.openai.com/v1"
      api_key: "${OPENAI_API_KEY}"

sensitive_words:
  - "敏感词1"
  - "敏感词2"
```

真实项目中，API Key 不应该直接写死在配置文件中，而应该通过环境变量读取。

---

## 10. 面试重点

### 10.1 为什么使用 POST + fetch streaming，而不是 EventSource？

浏览器原生 `EventSource` 只支持 GET，不适合携带复杂 JSON 请求体。AI 对话请求通常需要提交用户消息、模型参数和其他上下文信息，因此本项目使用 POST 请求提交数据，并通过 fetch 读取流式响应。服务端仍然按照 SSE 格式持续返回事件。

### 10.2 AI 回复为什么要先创建 assistant 占位消息？

模型生成不是瞬时操作，可能成功、失败、超时或被取消。先创建 `generating` 状态的 assistant 占位消息，可以记录 AI 回复从生成中到完成或失败的完整生命周期，也方便前端展示和后端排查问题。

### 10.3 为什么同一个会话同一时间只允许一个生成请求？

多轮对话依赖消息顺序。如果同一个 session 同时生成多个回复，可能导致上下文读取混乱、消息顺序错乱和回复不一致。因此使用 Redis 锁限制同一会话并发生成。

### 10.4 Redis 锁释放为什么要校验 value？

如果一个请求持有的锁过期后，另一个请求重新获得了锁，而第一个请求结束时直接删除 key，就可能误删第二个请求的锁。因此释放锁时必须比较 value 是否等于当前 request_id，并用 Lua 脚本保证比较和删除的原子性。

### 10.5 上下文如何维护？

系统从 MySQL 查询当前会话最近 N 条 `completed` 消息，按时间正序组装为模型上下文。`failed`、`cancelled` 和 `generating` 状态的消息不会进入上下文，避免模型读取不完整或错误内容。

### 10.6 Provider 为什么要抽象？

不同模型厂商的请求参数、鉴权方式和流式返回格式不同。业务层只依赖统一的 Provider 接口，具体模型调用逻辑放在 Provider 实现中。这样后续替换模型服务时，不需要改动聊天主流程。

### 10.7 Mock Provider 有什么价值？

Mock Provider 可以在没有真实 API Key 的情况下跑通注册、登录、创建会话、发送消息、流式返回、消息保存、状态更新和调用日志等完整链路。它也方便测试超时、取消和失败场景。

### 10.8 流式过程中的错误如何处理？

Provider 初始调用失败通过函数返回的 `error` 表达。流式生成过程中失败通过 `ChatChunk.Err` 表达。调用层收到错误后，需要向客户端发送 error 事件，并更新 assistant 消息状态和模型调用日志。

### 10.9 客户端断开连接怎么办？

请求 context 会被取消，Provider 需要监听 `ctx.Done()` 并停止生成。系统将 assistant 消息状态更新为 `cancelled`，模型调用日志记录为 `cancelled`，最后释放 Redis 锁。

### 10.10 失败重试为什么不适用于已经开始输出的流式生成？

如果流式内容已经返回给客户端，中途重试可能导致内容重复或顺序混乱。因此本项目只对 Provider 初始调用失败做有限重试，不对已经开始输出的流式过程做重试。

---

## 11. 项目完成标准

项目达到以下标准即视为完成：

1. 能启动 Go 后端服务；
2. 能通过 Docker Compose 启动 MySQL 和 Redis；
3. 用户能注册、登录，并通过 JWT 访问受保护接口；
4. 用户能创建、查询、详情查看和软删除会话；
5. 用户能查询模型列表，并只能选择后端白名单中的模型；
6. 用户能发送消息，系统能保存 user 消息和 assistant 占位消息；
7. 服务端能通过 POST + fetch streaming 返回流式内容；
8. Mock Provider 能模拟流式输出；
9. OpenAI 兼容 Provider 能接入真实模型服务；
10. user 消息和 assistant 消息能正确保存到 MySQL；
11. assistant 消息能正确流转 `generating`、`completed`、`failed`、`cancelled` 状态；
12. 同一个 session 同一时间只能有一个生成请求；
13. 能查询历史消息；
14. 能记录模型调用日志；
15. 能处理模型超时、失败重试和客户端取消；
16. 能进行简单敏感词过滤；
17. 能自动生成会话标题；
18. 能统计用户模型调用次数；
19. 能将对话导出为 Markdown；
20. README 中能清楚说明项目背景、技术栈、启动方式、接口示例和核心设计。

---

## 12. 项目总结

FlowChat 是一个偏 AI 应用后端方向的 Go 项目。

它和普通 CRUD 项目的区别在于：

- 有流式响应；
- 有多轮上下文；
- 有模型 Provider 抽象；
- 有生成状态管理；
- 有同会话并发控制；
- 有超时、取消和失败处理；
- 有模型调用日志和调用统计。

它和 KeyMeter 项目的关系是：

- KeyMeter 偏 API Key、额度扣减、幂等和调用计量；
- FlowChat 偏 AI 对话、上下文管理、流式响应和模型调用；
- 两个项目都使用 Go、MySQL、Redis、JWT、日志和后端工程化设计；
- 两个项目有技术联系，但业务主线不同。

因此，FlowChat 可以作为第二个 Go 后端实习项目，用来展示 AI 应用后端开发能力和完整工程链路设计能力。

