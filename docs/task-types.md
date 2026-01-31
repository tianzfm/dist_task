# 任务类型说明

## 概述

dist_task 支持多种任务类型，每种类型对应不同的执行器。

## RPC 任务

发起 RPC 调用（内部通过 HTTP 模拟）。

```json
{
  "id": "deduct",
  "task_name": "deduct",
  "description": "扣款",
  "config": {
    "service": "payment-service",
    "method": "deduct"
  }
}
```

**配置字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `service` | string | 是 | 服务地址 |
| `method` | string | 是 | 方法名 |

**输入参数：**

| 任务类型 | 必填参数 |
|----------|----------|
| `deduct` | `user_id`, `amount`, `order_id` |

## MQ 任务

发送 RocketMQ 消息。

```json
{
  "id": "notify",
  "task_name": "notify",
  "description": "发通知",
  "config": {
    "topic": "payment.completed"
  }
}
```

**配置字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topic` | string | 是 | MQ Topic |

**输入参数：**

| 任务类型 | 必填参数 |
|----------|----------|
| `notify` | `user_id`, `order_id`, `status` |

## HTTP 任务

发起 HTTP 请求。

```json
{
  "id": "http_request",
  "task_name": "http_request",
  "description": "调用外部服务",
  "config": {
    "url": "http://example.com/api",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer xxx"
    }
  }
}
```

**配置字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 是 | 请求地址 |
| `method` | string | 否 | HTTP 方法，默认 POST |
| `headers` | object | 否 | 请求头 |

**输入参数：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `body` | string | 请求体（可选） |

## DB 任务

执行数据库操作。

```json
{
  "id": "insert_order",
  "task_name": "db",
  "description": "插入订单",
  "config": {
    "operation": "insert",
    "table": "orders",
    "data": {
      "order_id": "${input.order_id}",
      "user_id": "${input.user_id}",
      "amount": "${input.amount}"
    }
  }
}
```

**配置字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `operation` | string | 是 | 操作类型：insert/update/delete |
| `table` | string | 是 | 表名 |
| `data` | object | 否 | 插入/更新的数据 |
| `where` | object | 否 | 条件（用于 update/delete） |

## 自定义任务类型

### 1. 注册任务定义

在 `pkg/taskdef/definition.go` 中添加：

```go
var TaskDefinitions = map[string]TaskDefinition{
    "my_task": {
        Name:        "我的任务",
        Type:        "rpc",
        Description: "自定义任务",
        InputFields: []Field{
            {Name: "param1", Type: "string", Required: true},
            {Name: "param2", Type: "int", Required: false, Default: 0},
        },
        Config: TaskConfig{
            Service: "my-service",
            Method:  "myMethod",
        },
    },
}
```

### 2. 添加执行器

在 `internal/engine/executor/executor.go` 中添加：

```go
type MyExecutor struct {
    client *http.Client
}

func (e *MyExecutor) Execute(ctx context.Context, config []byte, input map[string]interface{}) error {
    // 实现逻辑
}

func NewMyExecutor() *MyExecutor {
    return &MyExecutor{
        client: &http.Client{Timeout: 30 * time.Second},
    }
}
```

### 3. 注册执行器

在 `ExecutorFactory.Create()` 中注册：

```go
func (f *ExecutorFactory) Create(taskType string) (TaskExecutor, error) {
    switch taskType {
    // ... 已有类型
    case "my_task":
        return NewMyExecutor(), nil
    }
}
```
