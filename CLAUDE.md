# CLAUDE.md

This file provides guidance to Claude Code when working with this project.

## Project Overview

**dist_task** - A distributed transaction observability platform written in Go.

### Purpose
- Orchestrate distributed transaction workflows
- Monitor and track transaction execution
- Handle exceptions with manual or automatic retry strategies
- Provide observability and manual intervention capabilities

### Key Features
- Flow-based workflow orchestration (serial and parallel execution)
- Multiple task types: RPC, MQ, HTTP, DB
- Parameter validation and type conversion
- Exception tracking and retry mechanisms
- Manual intervention support
- Visualization Dashboard (in development)

## Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: MySQL 8.0 with GORM v2
- **Message Queue**: RocketMQ
- **Logging**: zerolog
- **Configuration**: TOML

## Project Structure

```
dist_task/
├── cmd/server/           # Application entry point
├── internal/
│   ├── api/handler/      # REST API handlers
│   ├── config/           # Configuration loading
│   ├── engine/           # Execution engine & scheduler
│   │   └── executor/     # Task executors (RPC, MQ, HTTP, DB)
│   ├── model/            # Data models
│   ├── repository/       # Data access layer (GORM)
│   └── retry/            # Automatic retry scheduler
├── pkg/
│   ├── errors/           # Error definitions
│   ├── logger/           # Logging wrapper (zerolog)
│   ├── mq/               # RocketMQ utilities
│   └── taskdef/          # Task definitions & validation
├── configs/              # Configuration files
├── migrations/           # Database migrations
├── docs/                 # Documentation
├── frontend/             # Web UI (future)
└── test/                 # Test data
```

## Development Standards

### Code Style
- Use `go fmt` for formatting
- Follow Effective Go conventions
- Use table-driven tests
- Add comments for public APIs

### Configuration
- Use TOML format for config files
- See `configs/app.toml.example` for template
- Environment variables for sensitive data

### Testing
- Run tests: `go test -v -race -cover ./...`
- Use table-driven test style
- Aim for >60% test coverage

### Building
```bash
# Build binary
go build -o bin/server ./cmd/server

# Run with Docker
docker-compose up -d

# Run tests
make test
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Application entry point |
| `internal/engine/scheduler.go` | Flow execution engine |
| `internal/engine/executor/executor.go` | Task executors |
| `internal/retry/scheduler.go` | Retry scheduler |
| `internal/api/handler/handler.go` | REST API handlers |
| `pkg/taskdef/definition.go` | Task definitions |

## Database Schema

| Table | Purpose |
|-------|---------|
| `task_group_flow` | Flow definitions (JSON format) |
| `task_group_instance` | Flow execution instances |
| `dist_task` | Individual task records |
| `exception_record` | Exception tracking |
| `execution_log` | Detailed execution logs |

## Common Tasks

### Adding a New Task Type

1. Define task in `pkg/taskdef/definition.go`:
```go
var TaskDefinitions = map[string]TaskDefinition{
    "my_task": {
        Name: "My Task",
        Type: "rpc",
        InputFields: []Field{
            {Name: "param1", Type: "string", Required: true},
        },
    },
}
```

2. Add executor in `internal/engine/executor/executor.go`

3. Register in `ExecutorFactory.Create()`

4. Add tests

### Adding API Endpoint

1. Add handler method in `internal/api/handler/handler.go`
2. Register route in routing setup
3. Add request/response structs
4. Add unit tests
5. Update API docs in `docs/api.md`

### Database Migrations

Create SQL files in `migrations/` directory following the pattern:
```
migrations/001_init_schema.sql
migrations/002_xxx.sql
```

## Git Workflow

1. Create feature branch: `git checkout -b feature/xxx`
2. Make changes and test
3. Commit: `git commit -m "feat/fix/docs: description"`
4. Push: `git push origin feature/xxx`
5. Create Pull Request

## API Endpoints

### Flow Management
- `POST /api/v1/flows` - Create flow
- `GET /api/v1/flows` - List flows
- `GET /api/v1/flows/:id` - Get flow details

### Transaction Management
- `POST /api/v1/transactions` - Start transaction
- `GET /api/v1/transactions/:id` - Get transaction status
- `POST /api/v1/transactions/:id/retry` - Retry failed transaction

### Exception Management
- `GET /api/v1/exceptions` - List exceptions
- `POST /api/v1/exceptions/:id/handle` - Handle exception
- `POST /api/v1/exceptions/:id/retry` - Schedule retry

### Utilities
- `GET /health` - Health check

## Roadmaps & Issues

See [docs/roadmap.md](docs/roadmap.md) for future plans.
See GitHub Issues for current tasks.

## Notes for AI Assistants

- When modifying core logic (engine/scheduler.go), ensure task dependencies are handled correctly
- Repository instances should be injected via constructor, not created inline
- All public APIs should have proper error handling and logging
- Update documentation when adding new features
- Write tests before or alongside implementation
