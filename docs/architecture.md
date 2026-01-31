# 架构设计

## 概述

dist_task 采用分层架构设计，主要分为以下几层：

```
┌─────────────────────────────────────────────────────────────────┐
│                        Presentation Layer                        │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │   Web UI     │  │  REST API    │  │   gRPC API (未来)      │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Application Layer                          │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │   Handler    │  │    Engine    │  │   Retry Scheduler      │ │
│  │   (请求处理)  │  │   (执行引擎)  │  │   (重试调度)           │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Domain Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │   Flow       │  │   Task       │  │   Exception            │ │
│  │   (流程)     │  │   (任务)     │  │   (异常)               │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Infrastructure Layer                       │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │   Executor   │  │  Repository  │  │   Logger               │ │
│  │   (执行器)   │  │   (数据访问)  │  │   (日志)               │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. Engine（执行引擎）

Engine 是系统的核心，负责解析 Flow 定义并调度任务执行。

**主要职责：**
- 解析 Flow JSON 定义
- 按依赖关系调度任务
- 参数传递和校验
- 异常捕获和记录

**关键文件：**
- `internal/engine/scheduler.go`

### 2. Executor（执行器）

执行器负责具体任务的执行，目前支持四种类型：

| 执行器 | 类型 | 用途 |
|--------|------|------|
| RPCExecutor | rpc | 发起 RPC 调用 |
| MQExecutor | mq | 发送 MQ 消息 |
| HTTPExecutor | http | 发起 HTTP 请求 |
| DBExecutor | db | 执行数据库操作 |

**关键文件：**
- `internal/engine/executor/executor.go`

### 3. Retry Scheduler（重试调度器）

负责自动重试失败的任务。

**特性：**
- 定时扫描待重试的异常
- 支持自定义重试策略（手动/自动）
- 指数退避重试间隔

**关键文件：**
- `internal/retry/scheduler.go`

### 4. Repository（数据访问层）

封装所有数据库操作，使用 GORM 作为 ORM。

**主要 Repository：**
- `FlowRepository` - Flow 定义管理
- `InstanceRepository` - 事务实例管理
- `TaskRepository` - 任务记录管理
- `ExceptionRepository` - 异常记录管理
- `LogRepository` - 日志管理

## 数据模型

### Flow（流程定义）

```go
type TaskGroupFlow struct {
    ID          string    // 唯一标识
    Name        string    // 名称
    Description string    // 描述
    FlowType    string    // 类型
    Version     int       // 版本号
    Definition  string    // JSON 格式的定义
    IsActive    bool      // 是否启用
    CreatedAt   time.Time // 创建时间
    UpdatedAt   time.Time // 更新时间
}
```

### Instance（事务实例）

```go
type TaskGroupInstance struct {
    ID          string     // 唯一标识
    FlowID      string     // 关联的 Flow ID
    Status      string     // 状态：pending/running/success/failed
    CreatedAt   time.Time  // 创建时间
    UpdatedAt   time.Time  // 更新时间
    CompletedAt *time.Time // 完成时间
}
```

### Task（任务记录）

```go
type DistTask struct {
    ID           string     // 唯一标识
    GroupID      string     // 所属实例 ID
    Name         string     // 名称
    Type         string     // 类型
    Status       string     // 状态
    Config       string     // 配置
    InputData    string     // 输入数据
    OutputData   string     // 输出数据
    ErrorMessage string     // 错误信息
    StartedAt    *time.Time // 开始时间
    CompletedAt  *time.Time // 完成时间
}
```

## API 设计

### 命名规范

- 资源名称使用单数形式（如：`flow`, `transaction`）
- 使用复数形式作为 API 路径（如：`/api/v1/flows`）
- HTTP 方法语义化（GET 查、POST 增、PUT 改、DELETE 删）

### 版本管理

API 版本通过 URL 路径管理：`/api/v1/`

### 响应格式

```json
{
    "code": 0,
    "message": "success",
    "data": {
        // 业务数据
    }
}
```

## 配置管理

使用 TOML 格式配置文件：

```toml
[app]
app_name = "dist_task"
host = "0.0.0.0"
port = 8080

[database]
host = "127.0.0.1"
port = 3306

[rocketmq]
namesrv = "127.0.0.1:9876"
```

## 扩展性设计

### Task 类型扩展

新增 Task 类型只需：

1. 在 `pkg/taskdef/definition.go` 中添加 Task 定义
2. 在 `internal/engine/executor/executor.go` 中添加对应 Executor
3. 在 ExecutorFactory 中注册

### 日志扩展

使用 zerolog，支持多种输出格式：
- JSON（生产环境）
- 控制台（开发环境）

### 存储扩展

通过 Repository 接口，可以轻松切换底层存储：
- MySQL（当前实现）
- PostgreSQL
- MongoDB
