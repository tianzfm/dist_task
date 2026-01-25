# dist_task

分布式事务可观测性平台

## 项目简介

一致性观测平台，专注于分布式事务的可观测性，提供流程编排、异常发现、观测定位能力。

## 核心功能

- 事务流编排：支持串行和并行任务执行
- 多种任务类型：RPC、MQ、HTTP、DB
- 参数解析：框架内置参数解析和校验能力
- 异常记录：完整的执行日志和异常追踪
- 人工兜底：支持异常的人工处理和重试

## 技术栈

- Go 1.21+
- MySQL 8.0
- RocketMQ
- GORM v2
- zerolog
- Gin

## 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0
- RocketMQ（可选）

### 配置

修改 `configs/app.toml` 中的数据库配置：

```toml
[database]
host = "127.0.0.1"
port = 3306
username = "root"
password = "your_password"
name = "dist_task"
```

### 数据库初始化

```bash
mysql -u root -p < migrations/001_init_schema.sql
```

### 启动服务

```bash
# 开发模式
go run cmd/server/main.go

# 构建
go build -o bin/server ./cmd/server

# 运行二进制
./bin/server
```

## API 文档

### 健康检查

```bash
GET /health
```

### 事务流管理

```bash
# 创建事务流
POST /api/v1/flows
# 列表查询
GET /api/v1/flows
# 详情
GET /api/v1/flows/:id
```

### 事务执行

```bash
# 启动事务流
POST /api/v1/transactions
# 查询详情
GET /api/v1/transactions/:id
```

### 异常管理

```bash
# 异常列表
GET /api/v1/exceptions
```

## 项目结构

```
dist_task/
├── cmd/                    # 应用入口
├── internal/              # 业务逻辑
│   ├── api/              # API 处理器
│   ├── config/           # 配置加载
│   ├── engine/           # 执行引擎
│   ├── model/            # 数据模型
│   └── repository/       # 数据访问层
├── pkg/                  # 公共包
├── configs/              # 配置文件
├── migrations/           # 数据库迁移
└── docs/                 # 文档
```

## License

MIT
