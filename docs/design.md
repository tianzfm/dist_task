# 一致性观测平台 - 设计文档

## 1. 项目概述

### 1.1 项目背景

分布式系统中，一致性问题是固有难题。原子处置能力、处置任务以及判罚写逻辑之间缺乏一致性观测和保障机制，导致：

- 一致性问题偶现，主动发现困难，依赖人工进线
- 无完整的事务状态变更日志，问题排查困难
- 回滚能力需适配开发，无标准化方案

### 1.2 项目定位

**一致性观测平台** - 专注于分布式事务的可观测性：

- **核心能力**：流程编排、异常发现、观测定位、人工兜底
- **设计原则**：可观测性优先，一致性保障可按需扩展
- **侵入性**：最小化，不强制要求业务方提供回滚机制

### 1.3 核心价值

| 价值点 | 说明 |
|-------|------|
| 效率提升 | 复杂处置能力接入效率高，问题排查快 |
| 稳定性保障 | 提供一致性观测手段，异常可追溯 |
| 灵活扩展 | 支持多种事务模式，可按需扩展 |

---

## 2. 技术架构

### 2.1 技术栈

| 类别 | 技术选型 | 说明 |
|-----|---------|------|
| 编程语言 | Go 1.21+ | 高性能、云原生友好 |
| 数据库 | MySQL 8.0 | 主存储，事务记录 |
| 消息队列 | RocketMQ | 消息驱动、异步处理 |
| ORM | GORM v2 | Go ORM 框架 |
| 日志 | zerolog | 结构化日志 |
| 配置格式 | TOML | 简洁配置格式 |

### 2.2 架构分层

```
┌─────────────────────────────────────────────────────────────┐
│ 接入层 │ API Gateway                                        │
│ 职责   │ 存在性校验、合法性校验、监控打点                     │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 管理端 │ Admin API                                          │
│ 职责   │ 事务流注册、异常查询、人工补偿等可视化平台           │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 服务端 │ Service API                                        │
│ 职责   │ 请求接收、合法性校验、配置解析、数据埋点统计         │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 引擎层 │ Execution Engine                                   │
│ 职责   │ 任务调度、状态变更、日志记录、重试执行               │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 执行层 │ Task Executor                                      │
│ 职责   │ RPC、MQ、HTTP、DB 执行（不感知业务逻辑）            │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 持久层 │ MySQL                                              │
│ 职责   │ 事务、任务、异常、日志记录存储                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 数据库设计

### 3.1 ER 图

```
task_group_flow (事务流定义)
    ↓
task_group_instance (事务流实例)
    ↓
dist_task (任务执行)
    ↓
execution_log (执行日志)
exception_record (异常记录)
```

### 3.2 表结构

#### 3.2.1 task_group_flow - 事务流定义表

```sql
CREATE TABLE task_group_flow (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    flow_type VARCHAR(50) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    definition JSON,
    is_active TINYINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    create_user VARCHAR(100) NOT NULL,
    updated_user VARCHAR(100) NOT NULL,
    UNIQUE KEY uk_name_ver (name, version)
);
```

**definition 结构：**

```json
{
  "name": "payment-flow",
  "description": "支付流程",
  "tasks": [
    {
      "id": "task_001",
      "task_name": "deduct",
      "description": "扣款",
      "config": {
        "service": "PaymentService",
        "method": "deduct"
      }
    },
    {
      "id": "task_002",
      "task_name": "notify",
      "depends_on": ["task_001"],
      "description": "发送通知",
      "config": {
        "topic": "payment.completed"
      }
    }
  ]
}
```

#### 3.2.2 task_group_instance - 事务流实例表

```sql
CREATE TABLE task_group_instance (
    id VARCHAR(64) PRIMARY KEY,
    flow_id VARCHAR(64) NOT NULL,
    status ENUM('pending', 'running', 'success', 'failed') DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    INDEX idx_flow_status (flow_id, status)
);
```

**幂等机制：**
- `id` 由业务方传入，作为幂等 key
- 同一 `id` 多次调用，返回相同结果

#### 3.2.3 dist_task - 任务执行表

```sql
CREATE TABLE dist_task (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type ENUM('rpc', 'mq', 'http', 'db') NOT NULL,
    status ENUM('pending', 'running', 'success', 'failed') DEFAULT 'pending',
    max_retry INT DEFAULT 3,
    retry_count INT DEFAULT 0,
    config JSON,
    input_data JSON,
    output_data JSON,
    error_message TEXT,
    error_stack TEXT,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    INDEX idx_group_status (group_id, status)
);
```

#### 3.2.4 exception_record - 异常记录表

```sql
CREATE TABLE exception_record (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL,
    group_name VARCHAR(255) NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    task_name VARCHAR(255) NOT NULL,
    error_type INT NOT NULL,
    error_code VARCHAR(100),
    error_message TEXT,
    stack_trace TEXT,
    retry_strategy VARCHAR(50) DEFAULT 'manual',
    retry_times INT DEFAULT 0,
    retry_max INT DEFAULT 3,
    retry_interval INT DEFAULT 60,
    retry_next_at TIMESTAMP NULL,
    handled BOOLEAN DEFAULT FALSE,
    handled_by VARCHAR(100),
    handled_at TIMESTAMP NULL,
    handled_remark TEXT,
    occurred_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_group_status (group_id, status),
    INDEX idx_retry_strategy (retry_strategy, handled)
);
```

#### 3.2.5 execution_log - 执行日志表

```sql
CREATE TABLE execution_log (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    group_id VARCHAR(64) NOT NULL,
    action ENUM('start', 'retry', 'success', 'failed', 'complete') NOT NULL,
    message TEXT,
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task (task_id)
);
```

---

## 4. Task 定义

### 4.1 内置 Task 类型

| 类型 | 说明 | 配置来源 |
|-----|------|---------|
| rpc | RPC 调用 | flow.config 定义 service/method |
| mq | 消息队列发送 | flow.config 定义 topic |
| http | HTTP 请求 | flow.config 定义 url/method/headers |
| db | 数据库操作 | flow.config 定义 operation/table |

### 4.2 Task 定义（硬编码）

```go
// internal/engine/task_definitions.go

var TaskDefinitions = map[string]TaskDefinition{
    "deduct": {
        Name:        "扣款",
        Type:        "rpc",
        Description: "从用户账户扣款",
        InputFields: []Field{
            {Name: "user_id", Type: "string", Required: true},
            {Name: "amount", Type: "int", Required: true},
            {Name: "order_id", Type: "string", Required: true},
        },
        Config: TaskConfig{
            Service: "PaymentService",
            Method:  "deduct",
        },
    },
    "notify": {
        Name:        "发送通知",
        Type:        "mq",
        Description: "发送支付完成通知",
        InputFields: []Field{
            {Name: "user_id", Type: "string", Required: true},
            {Name: "order_id", Type: "string", Required: true},
            {Name: "status", Type: "string", Required: true},
        },
        Config: TaskConfig{
            Topic: "payment.completed",
        },
    },
    "http_request": {
        Name:        "HTTP 请求",
        Type:        "http",
        Description: "发起 HTTP 请求",
        InputFields: []Field{
            {Name: "body", Type: "string", Required: false},
        },
        Config: TaskConfig{},
    },
}
```

### 4.3 参数类型

| 类型 | 说明 | 示例 |
|-----|------|------|
| string | 字符串 | "12345" |
| int | 整数 | 100 |
| bool | 布尔值 | true / false |

---

## 5. API 接口设计

### 5.1 事务流管理

| 方法 | 路径 | 说明 |
|-----|------|------|
| POST | /api/v1/flows | 创建事务流 |
| GET | /api/v1/flows | 列表查询 |
| GET | /api/v1/flows/:id | 详情 |
| PUT | /api/v1/flows/:id | 更新 |
| DELETE | /api/v1/flows/:id | 删除（软删） |
| POST | /api/v1/flows/:id/activate | 启用 |
| POST | /api/v1/flows/:id/deactivate | 停用 |

### 5.2 事务执行

| 方法 | 路径 | 说明 |
|-----|------|------|
| POST | /api/v1/transactions | 启动事务流（核心接口） |
| GET | /api/v1/transactions | 列表查询 |
| GET | /api/v1/transactions/:id | 详情 |
| GET | /api/v1/transactions/:id/tasks | 任务列表 |
| GET | /api/v1/transactions/:id/logs | 执行日志 |
| POST | /api/v1/transactions/:id/retry | 手动重试 |
| POST | /api/v1/transactions/:id/cancel | 取消 |

### 5.3 异常管理

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /api/v1/exceptions | 异常列表 |
| GET | /api/v1/exceptions/:id | 异常详情 |
| POST | /api/v1/exceptions/:id/handle | 人工处理 |
| POST | /api/v1/exceptions/:id/retry | 手动重试 |

### 5.4 健康检查

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /health | 健康检查 |

---

## 6. 执行引擎设计

### 6.1 执行流程

```
StartTransaction
    ↓
幂等检查（instance_id 是否存在）
    ↓
创建 instance，记录 status = pending
    ↓
启动执行引擎（异步）
    ↓
Engine.Execute:
    ↓
解析 flow，找到所有 task
    ↓
构建依赖图，拓扑排序
    ↓
按顺序执行任务（串行 + 并行）
    ↓
每个 task 执行：
    1. 解析参数（从 params 中提取）
    2. 校验参数（按 InputFields）
    3. 执行任务（调用对应执行器）
    4. 记录日志
    5. 更新状态
    ↓
所有 task 完成，更新 instance status
```

### 6.2 串行与并行

- **串行执行**：有 `depends_on` 依赖的任务
- **并行执行**：无依赖关系的任务自动并行

### 6.3 参数解析

**静态配置**：从 `flow.config` 中获取（url、service、topic 等）

**动态参数**：从 `params` 中按 task 分类传入

**示例：**

```json
// Flow definition
{
  "tasks": [{
    "id": "task_001",
    "task_name": "http_request",
    "config": {
      "url": "https://api.example.com/webhook",
      "method": "POST"
    }
  }]
}

// 业务调用
{
  "instance_id": "trans_001",
  "flow_id": "webhook-flow",
  "params": {
    "http_request": {
      "body": "{\"event\": \"payment_done\"}"
    }
  }
}

// 框架合并后执行
// url: https://api.example.com/webhook (from config)
// body: {"event": "payment_done"} (from params)
```

---

## 7. 项目结构

```
dist_task/
├── cmd/
│   └── server/
│       └── main.go              # 服务入口
├── internal/
│   ├── api/
│   │   └── handler/             # API 处理器
│   ├── config/                  # 配置加载
│   ├── engine/
│   │   ├── scheduler.go         # 调度器
│   │   ├── executor/            # 任务执行器
│   │   │   ├── rpc.go
│   │   │   ├── mq.go
│   │   │   ├── http.go
│   │   │   └── db.go
│   │   └── task_definitions.go  # Task 定义
│   ├── model/                   # 数据模型
│   ├── repository/              # 数据访问层
│   └── service/                 # 业务逻辑层
├── pkg/
│   ├── logger/                  # 日志封装
│   └── errors/                  # 错误定义
├── api/                         # API 定义
├── configs/                     # 配置文件
├── migrations/                  # 数据库迁移
├── scripts/                     # 工具脚本
├── test/                        # 测试数据
├── docs/                        # 文档
├── go.mod
├── go.sum
└── AGENTS.md                    # 开发规范
```

---

## 8. 后续规划

### 第一期（MVP）

- [x] 事务流编排
- [x] 任务执行（rpc, mq, http, db）
- [x] 参数解析与校验
- [x] 异常记录
- [x] 执行日志
- [x] 管理端 API

### 第二期

- [ ] 配置化 Task 定义（从代码中分离）
- [ ] 自动重试
- [ ] 监控告警
- [ ] 条件分支执行

### 第三期（可选）

- [ ] 回滚能力
- [ ] 强一致性事务
- [ ] 分布式部署

---

## 9. 参考资料

- 分布式事务解决方案：Saga、TCC、XA
- 本地消息表模式
- 阿里开源 Seata
- DTM 分布式事务框架
