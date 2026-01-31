# Contributing to dist_task

感谢你考虑为 dist_task 贡献代码！

## 如何贡献

### 1. 报告 Bug

通过 [GitHub Issues](https://github.com/dist_task/dist_task/issues) 报告：

- 描述问题
- 提供复现步骤
- 期望行为 vs 实际行为
- 日志和截图

### 2. 提出新功能

通过 [GitHub Issues](https://github.com/dist_task/dist_task/issues) 提出：

- 功能描述
- 使用场景
- 实现建议（可选）

### 3. 提交代码

1. Fork 项目
2. 创建特性分支：`git checkout -b feature/amazing-feature`
3. 提交更改：`git commit -m 'Add amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 创建 Pull Request

## 代码规范

### Go 规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go)
- 使用 `go fmt` 格式化代码
- 使用 `golint` 检查代码风格
- 注释公共 API

### 提交信息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type 类型：**
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具

**示例：**

```
feat(engine): Add task retry mechanism

Implement automatic retry for failed tasks with configurable
max attempts and interval.

Closes #123
```

## 开发环境

### 必需工具

- Go 1.21+
- MySQL 8.0
- Docker（可选）

### 本地开发

```bash
# 克隆代码
git clone https://github.com/your-username/dist_task.git
cd dist_task

# 安装依赖
go mod tidy

# 运行测试
go test -v ./...

# 启动服务
go run ./cmd/server/main.go
```

## 测试

提交代码前请确保：

```bash
# 运行所有测试
go test -v -race -cover ./...

# 检查代码
go vet ./...
```

## 许可证

贡献的代码将采用 MIT 许可证。
