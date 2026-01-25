package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"dist_task/internal/engine"
	"dist_task/pkg/logger"
)

type TaskExecutor interface {
	Execute(ctx context.Context, config []byte, input map[string]interface{}) error
}

type RPCExecutor struct{}

func (e *RPCExecutor) Execute(ctx context.Context, config []byte, input map[string]interface{}) error {
	var cfg engine.TaskConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse rpc config failed: %w", err)
	}

	logger.Info().
		Str("service", cfg.Service).
		Str("method", cfg.Method).
		Interface("input", input).
		Msg("RPC executor called")

	// TODO: 实现实际的 RPC 调用
	// 这里可以集成 gRPC、Dubbo 等 RPC 框架

	return nil
}

type MQExecutor struct{}

func (e *MQExecutor) Execute(ctx context.Context, config []byte, input map[string]interface{}) error {
	var cfg engine.TaskConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse mq config failed: %w", err)
	}

	message, _ := json.Marshal(input)

	logger.Info().
		Str("topic", cfg.Topic).
		RawJSON("message", message).
		Msg("MQ executor called")

	// TODO: 实现实际的 MQ 发送
	// 这里可以集成 RocketMQ

	return nil
}

type HTTPExecutor struct{}

func (e *HTTPExecutor) Execute(ctx context.Context, config []byte, input map[string]interface{}) error {
	var cfg engine.TaskConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse http config failed: %w", err)
	}

	var body io.Reader
	if bodyStr, ok := input["body"].(string); ok && bodyStr != "" {
		body = bytes.NewBufferString(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, body)
	if err != nil {
		return fmt.Errorf("create http request failed: %w", err)
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http request failed with status: %d", resp.StatusCode)
	}

	logger.Info().
		Str("url", cfg.URL).
		Str("method", cfg.Method).
		Int("status", resp.StatusCode).
		Msg("HTTP executor called")

	return nil
}

type DBExecutor struct{}

func (e *DBExecutor) Execute(ctx context.Context, config []byte, input map[string]interface{}) error {
	logger.Info().
		Interface("input", input).
		Msg("DB executor called")

	// TODO: 实现实际的 DB 操作
	// 这里可以集成 GORM 执行数据库操作

	return nil
}

func NewExecutor(taskType string) (TaskExecutor, error) {
	switch taskType {
	case "rpc":
		return &RPCExecutor{}, nil
	case "mq":
		return &MQExecutor{}, nil
	case "http":
		return &HTTPExecutor{}, nil
	case "db":
		return &DBExecutor{}, nil
	default:
		return nil, fmt.Errorf("unsupported task type: %s", taskType)
	}
}
