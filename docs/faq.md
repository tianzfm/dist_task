# 常见问题

## Q1: 启动服务时报数据库连接错误

**问题：** `dial tcp 127.0.0.1:3306: connect: connection refused`

**解决：**
1. 确认 MySQL 已启动
2. 检查配置文件中的数据库连接信息
3. 确保数据库用户有权限访问

## Q2: Flow 创建成功但执行失败

**问题：** 返回 `flow not found`

**解决：**
1. 检查 `flow_id` 是否正确
2. 确认 Flow 已创建（使用 `GET /api/v1/flows` 查看）

## Q3: 重试策略不生效

**问题：** 任务失败后没有自动重试

**解决：**
1. 检查 Flow 定义中的 `retry` 配置
2. 确认 `strategy` 为 `auto`
3. 检查重试调度器是否启动

## Q4: RocketMQ 连接失败

**问题：** `MQ connect failed`

**解决：**
1. 确认 RocketMQ 服务已启动
2. 检查 `nameserver` 地址配置
3. 如果不需要 MQ 功能，可以注释相关配置

## Q5: 如何查看执行日志

**方法：**
1. 通过 API 查询：`GET /api/v1/transactions/:id`
2. 查看数据库 `execution_log` 表
3. 查看服务日志文件（配置 output 为 file）

## Q6: 支持哪些数据库

**当前：** MySQL 8.0

**未来计划：** PostgreSQL、MongoDB

## Q7: 如何添加自定义任务类型

参考[任务类型说明](task-types.md#自定义任务类型)

## Q8: 任务之间的数据如何传递

**当前版本：** 通过全局参数传递

**未来版本：** 支持上下文传递（Context Passing）

## Q9: 如何实现条件执行

**当前方案：** 通过创建不同的 Flow 实现

**未来计划：** 内置 `condition` 和 `switch` 任务类型

## Q10: 服务性能如何

**基准测试：**
- 单实例可处理 ~1000 个并发任务
- 任务平均执行时间 < 1s（不含实际业务调用）

**优化建议：**
1. 增加数据库连接池
2. 使用 Redis 缓存 Flow 定义
3. 水平扩展多实例
