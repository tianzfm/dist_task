# Flow 定义指南

## 概述

Flow 是 dist_task 的核心概念，用于定义业务流程。每个 Flow 包含多个 Task，Task 之间可以有依赖关系。

## Flow 结构

```json
{
  "name": "payment_flow",
  "description": "支付流程",
  "tasks": [
    {
      "id": "task_1",
      "task_name": "deduct",
      "description": "扣款",
      "depends_on": [],
      "config": {},
      "retry": {}
    }
  ]
}
```

## 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | Flow 名称 |
| `description` | string | 否 | 描述 |
| `tasks` | array | 是 | 任务列表 |

### Task 字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 任务唯一标识 |
| `task_name` | string | 是 | 任务类型名称 |
| `description` | string | 否 | 任务描述 |
| `depends_on` | array | 否 | 依赖的任务 ID 列表 |
| `config` | object | 是 | 任务配置 |
| `retry` | object | 否 | 重试策略 |

## 完整示例

### 串行执行流程

```json
{
  "name": "order_flow",
  "description": "订单处理流程",
  "tasks": [
    {
      "id": "deduct",
      "task_name": "deduct",
      "description": "扣款",
      "depends_on": [],
      "config": {
        "service": "payment-service",
        "method": "deduct"
      },
      "retry": {
        "strategy": "auto",
        "max_attempts": 3,
        "interval": 5
      }
    },
    {
      "id": "inventory",
      "task_name": "deduct_inventory",
      "description": "扣库存",
      "depends_on": ["deduct"],
      "config": {
        "service": "inventory-service",
        "method": "deduct"
      }
    },
    {
      "id": "notify",
      "task_name": "notify",
      "description": "发通知",
      "depends_on": ["inventory"],
      "config": {
        "topic": "order.completed"
      }
    }
  ]
}
```

执行顺序：`deduct` → `inventory` → `notify`

### 并行执行流程

```json
{
  "name": "parallel_flow",
  "description": "并行处理流程",
  "tasks": [
    {
      "id": "task_a",
      "task_name": "http_request",
      "description": "任务 A",
      "depends_on": [],
      "config": {
        "url": "http://service-a/process",
        "method": "POST"
      }
    },
    {
      "id": "task_b",
      "task_name": "http_request",
      "description": "任务 B",
      "depends_on": [],
      "config": {
        "url": "http://service-b/process",
        "method": "POST"
      }
    },
    {
      "id": "task_c",
      "task_name": "http_request",
      "description": "任务 C（等待 A 和 B 完成）",
      "depends_on": ["task_a", "task_b"],
      "config": {
        "url": "http://service-c/process",
        "method": "POST"
      }
    }
  ]
}
```

执行顺序：`task_a` 和 `task_b` 并行执行 → `task_c`

## 重试策略

```json
"retry": {
  "strategy": "auto",     // manual / auto / no_retry
  "max_attempts": 3,      // 最大重试次数
  "interval": 5           // 重试间隔（秒）
}
```

| 策略 | 说明 |
|------|------|
| `manual` | 失败后不自动重试，等待人工处理 |
| `auto` | 按配置自动重试 |
| `no_retry` | 不重试 |

## 参数传递

### 全局参数

启动事务时传入的参数会被传递给所有 Task：

```json
{
  "params": {
    "deduct": {
      "user_id": "user_001",
      "amount": 100
    },
    "notify": {
      "user_id": "user_001",
      "message": "支付成功"
    }
  }
}
```

### 上下文传递（开发中）

后续版本将支持 Task 间传递数据：

```json
{
  "tasks": [
    {
      "id": "task_1",
      "task_name": "deduct",
      "output": "payment_result"
    },
    {
      "id": "task_2",
      "task_name": "notify",
      "input_from": {
        "payment_result": "$.task_1.result"
      }
    }
  ]
}
```

## 最佳实践

1. **Task ID 命名**：使用有意义的名称，如 `deduct_payment`、`send_notification`
2. **依赖关系**：尽量减少循环依赖，保持流程清晰
3. **重试策略**：根据任务重要性设置合适的策略
4. **错误处理**：关键任务建议使用 `manual` 策略，便于人工干预
