# 部署指南

## 方式一：直接运行

### 1. 编译

```bash
go build -o bin/server ./cmd/server
```

### 2. 运行

```bash
./bin/server
```

### 3. 指定配置文件

```bash
./bin/server -config /path/to/app.toml
```

---

## 方式二：使用 Docker

### 1. 构建镜像

```bash
docker build -t dist_task:latest .
```

### 2. 运行容器

```bash
docker run -d \
  --name dist_task \
  -p 8080:8080 \
  -v $(pwd)/configs/app.toml:/app/configs/app.toml \
  dist_task:latest
```

---

## 方式三：使用 Docker Compose

### 1. 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  dist_task:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./configs/app.toml:/app/configs/app.toml
    depends_on:
      - mysql
    environment:
      - DATABASE_HOST=mysql
      - DATABASE_PORT=3306
      - DATABASE_NAME=dist_task
      - DATABASE_USER=root
      - DATABASE_PASSWORD=root123

  mysql:
    image: mysql:8.0
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: dist_task
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  mysql_data:
```

### 2. 启动

```bash
docker-compose up -d
```

### 3. 初始化数据库

```bash
docker-compose exec mysql mysql -uroot -proot123 dist_task < migrations/001_init_schema.sql
```

---

## 方式四：Kubernetes 部署

### 1. 创建 ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: dist-task-config
data:
  app.toml: |
    [database]
    host = mysql-service
    port = 3306
    name = dist_task
```

### 2. 创建 Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dist-task
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dist-task
  template:
    metadata:
      labels:
        app: dist-task
    spec:
      containers:
      - name: dist-task
        image: dist_task:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /app/configs
      volumes:
      - name: config
        configMap:
          name: dist-task-config
```

---

## 生产环境配置建议

### 数据库配置

```toml
[database]
host = "10.0.0.1"
port = 3306
username = "dist_task"
password = "your_strong_password"
name = "dist_task"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = 300
```

### 日志配置

```toml
[log]
level = "info"      # 生产环境用 info
format = "json"     # 生产环境用 json
output = "stdout"
```

### RocketMQ 配置

```toml
[rocketmq]
namesrv = "10.0.0.2:9876"
producer_group = "dist_task_producer"
consumer_group = "dist_task_consumer"
```

---

## 监控配置

建议配置以下监控：

1. **Prometheus 指标**：暴露 `/metrics` 端点
2. **健康检查**：`/health` 端点用于 K8s liveness probe
3. **日志收集**：集成 ELK 或 Loki
4. **链路追踪**：集成 Jaeger 或 Zipkin
