# Consistency-Observability-Platform

## 项目概述
分布式事务可观测性平台，提供流程编排、异常发现、观测定位能力，支持人工兜底。

## 技术栈
- 语言：Go 1.21+
- 数据库：MySQL 8.0
- 消息队列：RocketMQ
- ORM：GORM v2
- 日志：zerolog
- 配置：TOML

## 构建与测试

### 常用命令
```bash
# 下载依赖
go mod tidy

# 本地开发启动
go run cmd/server/main.go

# 构建二进制
go build -o bin/server ./cmd/server

# 运行测试
go test ./... -v

# 运行单测试（指定文件/函数）
go test ./internal/task -run TestTaskExecutor -v

# 代码检查
golangci-lint run ./...

# 代码格式化
gofmt -w .
gci write ./...

# 数据库迁移
go run scripts/migrate/main.go up
```

## 项目结构
```
dist_task/
├── cmd/                    # 应用入口
│   └── server/            # 服务入口
├── internal/              # 业务逻辑（不对外暴露）
│   ├── api/              # API handler
│   ├── config/           # 配置加载
│   ├── service/          # 业务逻辑层
│   ├── repository/       # 数据访问层
│   ├── model/            # 数据模型
│   └── task/             # 任务引擎
├── pkg/                  # 可复用公共包
│   ├── logger/           # 日志封装
│   ├── mq/               # RocketMQ 封装
│   └── errors/           # 错误定义
├── api/                  # API 定义（OpenAPI）
├── configs/              # 配置文件
├── migrations/           # SQL 迁移文件
├── scripts/              # 工具脚本
├── test/                 # 测试数据
├── docs/                 # 文档
├── go.mod
├── go.sum
└── AGENTS.md
```

## 代码规范

### 命名规范
- **Go 文件/变量/函数**：CamelCase（如 `taskExecutor`、`GetGroupByID`）
- **数据库表名**：snake_case（如 `task_group_flow`）
- **数据库列名**：snake_case（如 `created_at`）
- **常量**：UPPER_SNAKE（如 `MAX_RETRY_COUNT`）
- **配置项**：snake_case（如 `db_max_open_conns`）
- **包名**：简短有意义（如 `repository`、`mq`）

### Import 排序
```go
import (
    // 标准库
    "context"
    "fmt"

    // 第三方
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    // 内部包
    "dist_task/internal/model"
    "dist_task/pkg/logger"
)
```

### 错误处理
```go
// 1. 使用 wrap 保留堆栈
if err != nil {
    return fmt.Errorf("failed to get task: %w", err)
}

// 2. 定义 sentinel error
var ErrTaskNotFound = errors.New("task not found")

// 3. 业务错误使用错误码
type BizError struct {
    Code    string
    Message string
}

// 4. 快速判断错误类型
if errors.Is(err, ErrTaskNotFound) {
    // 处理
}
```

### 日志规范
```go
// 使用 zerolog
log.Info().Str("task_id", id).Int("retry", count).Msg("task retry")

// 错误日志
log.Error().Err(err).Str("group_id", groupID).Msg("task failed")

// 敏感数据脱敏
log.Info().Str("phone", maskPhone(phone)).Send()
```

### Context 使用
```go
// Context 作为第一个参数
func GetTask(ctx context.Context, id string) (*Task, error)

// 不将 Context 放入结构体
type Service struct {
    db *gorm.DB
    // 不要放 ctx context.Context
}
```

### 数据库操作
```go
// 表名：复数形式或按业务命名（参考设计文档）
func (TaskGroupFlow) TableName() string {
    return "task_group_flow"
}

// 主键：统一使用 VARCHAR(64) + UUID
type BaseModel struct {
    ID        string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// 事务
db.Transaction(func(tx *gorm.DB) error {
    // 业务逻辑
    return nil
})
```

### API 设计
```go
// RESTful 风格
// GET    /api/v1/tasks          # 列表
// GET    /api/v1/tasks/:id      # 详情
// POST   /api/v1/tasks          # 创建
// PUT    /api/v1/tasks/:id      # 更新
// DELETE /api/v1/tasks/:id      # 删除

// 响应格式
{
    "code": 0,
    "message": "success",
    "data": {}
}
```

### RocketMQ 规范
```go
// 生产者：统一封装，支持重试
producer := mq.NewProducer()

// 消费者：幂等处理 + Graceful Shutdown
consumer := mq.NewConsumer(handler)
go consumer.Start()
```

### 测试要求
```go
// 单元测试使用 testify
func TestTaskExecutor_Execute(t *testing.T) {
    // Given
    // When
    // Then
}

// Mock 使用 mockgen
// 覆盖率：核心逻辑 > 80%
```

### 代码注释
```go
// GetTaskByID 根据任务ID获取任务详情
func GetTaskByID(ctx context.Context, id string) (*Task, error) {
    // 复杂逻辑需内部注释
}
```

### Git 提交规范
```
feat: 新功能
fix: 修复 bug
docs: 文档更新
refactor: 重构
test: 测试
chore: 构建/工具

示例：
feat(task): 添加任务重试机制
fix(repository): 修复事务泄漏问题
```
