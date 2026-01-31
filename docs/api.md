# API 文档

## 概述

dist_task 提供 RESTful API 进行所有操作。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **版本**: v1
- **Content-Type**: application/json

## 响应格式

```json
{
    "code": 0,
    "message": "success",
    "data": {}
}
```

**状态码说明：**

| code | 说明 |
|------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 健康检查

### GET /health

检查服务健康状态。

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "status": "ok"
    }
}
```

---

## Flow 管理

### POST /api/v1/flows

创建 Flow。

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Flow 名称 |
| description | string | 否 | 描述 |
| flow_type | string | 是 | 类型 |
| definition | string | 是 | Flow 定义（JSON 字符串） |
| create_user | string | 是 | 创建人 |

**请求示例：**

```bash
curl -X POST http://localhost:8080/api/v1/flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payment_flow",
    "description": "支付流程",
    "flow_type": "payment",
    "definition": "{\"name\":\"payment_flow\",\"tasks\":[]}",
    "create_user": "admin"
  }'
```

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "abc123",
        "name": "payment_flow",
        "created_at": "2024-01-31T10:00:00Z"
    }
}
```

### GET /api/v1/flows

获取 Flow 列表。

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [
            {
                "id": "abc123",
                "name": "payment_flow",
                "description": "支付流程"
            }
        ],
        "pagination": {
            "page": 1,
            "page_size": 20,
            "total": 1
        }
    }
}
```

### GET /api/v1/flows/:id

获取 Flow 详情。

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "abc123",
        "name": "payment_flow",
        "definition": "{\"name\":\"payment_flow\",\"tasks\":[]}"
    }
}
```

---

## 事务管理

### POST /api/v1/transactions

启动事务。

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | string | 是 | 实例 ID（业务方指定） |
| flow_id | string | 是 | Flow ID |
| params | object | 否 | 流程参数 |

**请求示例：**

```bash
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "order_001",
    "flow_id": "abc123",
    "params": {
      "deduct": {
        "user_id": "user_001",
        "amount": 100
      }
    }
  }'
```

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "instance_id": "order_001",
        "flow_id": "abc123",
        "status": "pending"
    }
}
```

### GET /api/v1/transactions/:id

获取事务状态。

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "instance_id": "order_001",
        "flow_id": "abc123",
        "status": "success",
        "tasks": [
            {
                "id": "order_001_deduct",
                "name": "扣款",
                "status": "success"
            }
        ],
        "created_at": "2024-01-31T10:00:00Z",
        "completed_at": "2024-01-31T10:00:10Z"
    }
}
```

### POST /api/v1/transactions/:id/retry

重试失败的事务。

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "instance_id": "order_001",
        "status": "pending"
    }
}
```

---

## 异常管理

### GET /api/v1/exceptions

获取异常列表。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| handled | bool | 是否已处理 |

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [
            {
                "id": 1,
                "task_id": "order_001_deduct",
                "task_name": "扣款",
                "error_message": "RPC call failed",
                "handled": false
            }
        ],
        "pagination": {
            "page": 1,
            "page_size": 20,
            "total": 1
        }
    }
}
```

### POST /api/v1/exceptions/:id/handle

标记异常为已处理。

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| remark | string | 否 | 处理备注 |

**请求示例：**

```bash
curl -X POST http://localhost:8080/api/v1/exceptions/1/handle \
  -H "Content-Type: application/json" \
  -d '{
    "remark": "人工确认问题已解决"
  }'
```

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "exception_id": 1,
        "handled": true
    }
}
```

### POST /api/v1/exceptions/:id/retry

安排异常重试。

**响应示例：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "exception_id": 1,
        "retry_scheduled": true,
        "retry_next_at": "2024-01-31T10:05:00Z"
    }
}
```

---

## 统计接口（开发中）

### GET /api/v1/stats/overview

获取概览统计。

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "total_transactions": 1000,
        "success_rate": 0.95,
        "avg_duration": 5.2,
        "exceptions_today": 5
    }
}
```

### GET /api/v1/stats/flows/:id

获取指定 Flow 的统计信息。

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "flow_id": "abc123",
        "total_runs": 500,
        "success_count": 480,
        "failed_count": 20
    }
}
```
