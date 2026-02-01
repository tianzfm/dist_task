# 问题排查记录

本文档记录了项目部署过程中遇到的两个典型问题及其排查过程。

---

## 问题 5：SQL 文件语法错误

### 问题现象

初始化数据库时执行 SQL 文件报错：

```bash
$ docker exec -i dist_task_mysql mysql -uroot -proot123 dist_task < migrations/001_init_schema.sql

mysql: [Warning] Using a password on the command line interface can be insecure.
ERROR 1072 (42000) at line 47: Key column 'status' doesn't exist in table
```

错误发生在第 47 行，提示 `status` 列不存在。

### 排查思路

1. **定位错误行**：SQL 文件第 47 行是 `exception_record` 表的建表语句
2. **分析索引定义**：检查该表的 `INDEX idx_group_status (group_id, status)` 索引
3. **对比表结构**：发现 `exception_record` 表定义中确实没有 `status` 列
4. **确认是 SQL 编写错误**：索引引用了不存在的列

### 排查过程

#### 步骤 1：读取 SQL 文件，定位问题行

```bash
# 查看 SQL 文件内容，重点关注第 47 行附近
cat migrations/001_init_schema.sql | head -70

# 或使用 sed 查看指定行
sed -n '40,70p' migrations/001_init_schema.sql
```

输出显示第 67 行有问题：

```sql
CREATE TABLE exception_record (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    -- ... 其他列定义 ...
    occurred_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_group_status (group_id, status),  -- 问题所在！
    INDEX idx_retry_strategy (retry_strategy, handled)
);
```

#### 步骤 2：对比其他表的索引写法

发现 `task_group_instance` 和 `dist_task` 表有 `status` 列，所以它们的索引是合理的：

```sql
CREATE TABLE task_group_instance (
    -- ...
    status ENUM('pending', 'running', 'success', 'failed') DEFAULT 'pending',
    INDEX idx_flow_status (flow_id, status)  -- 正确：status 列存在
);
```

#### 步骤 3：确认 `exception_record` 表没有 status 列

该表的设计是记录异常信息，不需要状态字段，重试策略由 `retry_strategy` 等字段控制。

### 最终结果

修复后的 SQL 文件：

```sql
CREATE TABLE exception_record (
    -- ... 列定义 ...
    occurred_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_group_id (group_id),                    -- 改为只索引 group_id
    INDEX idx_retry_strategy (retry_strategy, handled)
);
```

然后重新初始化数据库：

```bash
# 删除旧表
docker exec dist_task_mysql mysql -uroot -proot123 -e "USE dist_task; DROP TABLE IF EXISTS execution_log, exception_record, dist_task, task_group_instance, task_group_flow;"

# 重新执行 SQL
docker exec -i dist_task_mysql mysql -uroot -proot123 dist_task < migrations/001_init_schema.sql
```

### 用到的命令

| 命令 | 用途 |
|------|------|
| `cat file | head -n` | 查看文件前 n 行 |
| `sed -n 'start,endp'` | 查看文件指定行范围 |
| `docker exec <container> mysql -uuser -ppass -e "SQL"` | 在容器内执行 MySQL 命令 |

---

## 问题 6：配置文件格式错误导致服务无法访问

### 问题现象

服务启动后健康检查失败：

```bash
$ curl http://localhost:8080/health
curl: (52) Empty reply from server
```

容器日志显示服务已启动：

```bash
$ docker logs dist_task --tail 10

[GIN-debug] Listening and serving HTTP on :0  # 注意这里端口是 :0！
```

检查容器端口映射正常，但无法访问。

### 排查思路

1. **检查容器是否正常运行**：`docker ps` 确认容器 Up 状态
2. **检查端口映射**：确认宿主机 8080 映射到容器端口
3. **检查容器内监听端口**：确认服务是否在监听
4. **检查配置解析**：确认配置文件是否正确解析

### 排查过程

#### 步骤 1：确认容器运行状态

```bash
# 查看容器运行状态和端口映射
$ docker ps | grep dist_task

305952a94137   dist_task_dist_task   "./server"   39 minutes ago   Up 23 seconds (health: starting)   0.0.0.0:8080->8080/tcp
```

容器正常运行，端口映射正确。

#### 步骤 2：检查容器内监听的端口

```bash
# 进入容器检查监听端口
$ docker exec dist_task netstat -tlnp

Active Internet connections (only servers)
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program
      0 tcp        0127.0.0.11:45575        0.0.0.0:*               LISTEN      -
tcp        0      0 :::37499                :::*                    LISTEN      1/server   # 端口是 37499！
```

发现问题：服务监听在 `37499` 端口，而不是配置文件中的 `8080`！

#### 步骤 3：检查配置文件内容

```bash
# 查看容器内的配置文件
$ docker exec dist_task cat /app/configs/app.toml

# Application
app_name = "dist_task"
host = "0.0.0.0"
port = 8080   # 配置正确，但没生效
```

配置文件看起来正确，但为什么端口是 37499？

#### 步骤 4：检查启动日志

```bash
$ docker logs dist_task 2>&1 | head -30

2026/02/01 11:19:20 server starting on :0   # Port=0 导致随机端口！
```

`server starting on :0` 说明 `cfg.App.Port` 的值是 0。

#### 步骤 5：检查代码中的配置解析逻辑

查看 `internal/config/config.go`：

```go
type Config struct {
    App      AppConfig      `toml:"app"`      // 期望在 [app] 小节下
    Database DatabaseConfig `toml:"database"`
    // ...
}

type AppConfig struct {
    Name string `toml:"app_name"`
    Host string `toml:"host"`
    Port int    `toml:"port"`
}
```

代码期望配置在 `[app]` 小节下解析。

#### 步骤 6：对比配置文件结构

配置文件 `configs/app.toml`：

```toml
# Application
app_name = "dist_task"     # 顶层，无小节标记
host = "0.0.0.0"
port = 8080

[database]                  # 有小节标记
host = "mysql"
```

**问题根源**：`app_name`, `host`, `port` 放在**顶层**，没有 `[app]` 小节标记，导致 toml 解析时这些字段无法映射到 `AppConfig` 结构体，`Port` 取默认值 0。

### 最终结果

修复配置文件，添加 `[app]` 小节：

```toml
# Application
[app]                       # 添加小节标记
app_name = "dist_task"
host = "0.0.0.0"
port = 8080

[database]
host = "mysql"
```

重启服务验证：

```bash
$ docker restart dist_task && sleep 3

$ curl http://localhost:8080/health
{"code":0,"data":{"status":"ok"},"message":"success"}
```

### 用到的命令

| 命令 | 用途 |
|------|------|
| `docker ps` | 查看运行中的容器 |
| `docker ps | grep name` | 过滤查看指定容器 |
| `docker exec <container> netstat -tlnp` | 查看容器内网络连接和监听端口 |
| `docker exec <container> cat /path/file` | 查看容器内文件内容 |
| `docker logs <container> --tail n` | 查看容器最近 n 行日志 |
| `docker restart <container>` | 重启容器 |

### 排查技巧总结

| 症状 | 可能原因 | 排查命令 |
|------|----------|----------|
| 服务无法访问 | 端口未正确监听 | `docker exec container netstat -tlnp` |
| 配置未生效 | 配置文件格式错误 | `docker exec container cat config.toml` |
| 随机端口 | 配置解析失败 | `docker logs container | grep "starting"` |
| SQL 执行失败 | 表/列定义错误 | `sed -n 'start,endp' file.sql` |
