package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dist_task/internal/config"
	"dist_task/pkg/logger"
	"dist_task/pkg/taskdef"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"gorm.io/gorm"
)

type TaskExecutor interface {
	Execute(ctx context.Context, config []byte, input map[string]interface{}) error
}

type RPCExecutor struct {
	client *http.Client
}

func NewRPCExecutor() *RPCExecutor {
	return &RPCExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *RPCExecutor) Execute(ctx context.Context, cfgBytes []byte, input map[string]interface{}) error {
	var cfg taskdef.TaskConfig
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return fmt.Errorf("parse rpc config failed: %w", err)
	}

	if cfg.Service == "" || cfg.Method == "" {
		return fmt.Errorf("rpc config incomplete: service=%s, method=%s", cfg.Service, cfg.Method)
	}

	payload := map[string]interface{}{
		"method": cfg.Method,
		"params": input,
	}
	bodyBytes, _ := json.Marshal(payload)

	url := fmt.Sprintf("http://%s/rpc", cfg.Service)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create rpc request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("rpc call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rpc call failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.Info().
		Str("service", cfg.Service).
		Str("method", cfg.Method).
		Interface("input", input).
		Int("status", resp.StatusCode).
		Msg("RPC executor completed")

	return nil
}

type MQExecutor struct {
	producer rocketmq.Producer
}

func NewMQExecutor() (*MQExecutor, error) {
	cfg := config.GlobalConfig
	if cfg == nil {
		return nil, fmt.Errorf("config not initialized")
	}

	p, err := rocketmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{cfg.RocketMQ.NameServer})),
		producer.WithRetry(2),
	)
	if err != nil {
		return nil, fmt.Errorf("create mq producer failed: %w", err)
	}

	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("start mq producer failed: %w", err)
	}

	return &MQExecutor{producer: p}, nil
}

func (e *MQExecutor) Execute(ctx context.Context, cfgBytes []byte, input map[string]interface{}) error {
	var cfg taskdef.TaskConfig
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return fmt.Errorf("parse mq config failed: %w", err)
	}

	if cfg.Topic == "" {
		return fmt.Errorf("mq topic is required")
	}

	messageBody, _ := json.Marshal(input)
	msg := primitive.NewMessage(cfg.Topic, messageBody)

	result, err := e.producer.SendSync(ctx, msg)
	if err != nil {
		return fmt.Errorf("send mq message failed: %w", err)
	}

	logger.Info().
		Str("topic", cfg.Topic).
		RawJSON("message", messageBody).
		Str("msg_id", result.MsgID).
		Msg("MQ executor completed")

	return nil
}

type HTTPExecutor struct {
	client *http.Client
}

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *HTTPExecutor) Execute(ctx context.Context, cfgBytes []byte, input map[string]interface{}) error {
	var cfg taskdef.TaskConfig
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return fmt.Errorf("parse http config failed: %w", err)
	}

	if cfg.URL == "" {
		return fmt.Errorf("http url is required")
	}

	method := "POST"
	if cfg.Method != "" {
		method = strings.ToUpper(cfg.Method)
	}

	var body io.Reader
	if bodyStr, ok := input["body"].(string); ok && bodyStr != "" {
		body = bytes.NewBufferString(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, body)
	if err != nil {
		return fmt.Errorf("create http request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, _ := io.ReadAll(resp.Body)

	logger.Info().
		Str("url", cfg.URL).
		Str("method", method).
		Int("status", resp.StatusCode).
		Int("body_size", len(respBody)).
		Msg("HTTP executor completed")

	return nil
}

type DBExecutor struct {
	db *gorm.DB
}

func NewDBExecutor(db *gorm.DB) *DBExecutor {
	return &DBExecutor{db: db}
}

type DBConfig struct {
	Operation string                 `json:"operation"`
	Table     string                 `json:"table"`
	Data      map[string]interface{} `json:"data"`
	Where     map[string]interface{} `json:"where"`
}

func (e *DBExecutor) Execute(ctx context.Context, cfgBytes []byte, input map[string]interface{}) error {
	var cfg DBConfig
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return fmt.Errorf("parse db config failed: %w", err)
	}

	if cfg.Operation == "" || cfg.Table == "" {
		return fmt.Errorf("db config incomplete: operation=%s, table=%s", cfg.Operation, cfg.Table)
	}

	switch strings.ToLower(cfg.Operation) {
	case "insert":
		return e.insert(ctx, cfg.Table, cfg.Data)
	case "update":
		return e.update(ctx, cfg.Table, cfg.Data, cfg.Where)
	case "delete":
		return e.delete(ctx, cfg.Table, cfg.Where)
	default:
		return fmt.Errorf("unsupported db operation: %s", cfg.Operation)
	}
}

func (e *DBExecutor) insert(ctx context.Context, table string, data map[string]interface{}) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("insert data is required")
	}

	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for k, v := range data {
		columns = append(columns, k)
		placeholders = append(placeholders, "?")
		values = append(values, v)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ","), strings.Join(placeholders, ","))

	result := e.db.WithContext(ctx).Exec(query, values...)
	if result.Error != nil {
		return fmt.Errorf("insert failed: %w", result.Error)
	}

	logger.Info().
		Str("table", table).
		Int("affected", int(result.RowsAffected)).
		Msg("DB insert completed")

	return nil
}

func (e *DBExecutor) update(ctx context.Context, table string, data, where map[string]interface{}) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("update data is required")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)+len(where))

	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", k))
		values = append(values, v)
	}

	whereClauses := make([]string, 0, len(where))
	for k, v := range where {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", k))
		values = append(values, v)
	}

	query := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setClauses, ","))
	if len(whereClauses) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(whereClauses, " AND "))
	}

	result := e.db.WithContext(ctx).Exec(query, values...)
	if result.Error != nil {
		return fmt.Errorf("update failed: %w", result.Error)
	}

	logger.Info().
		Str("table", table).
		Int("affected", int(result.RowsAffected)).
		Msg("DB update completed")

	return nil
}

func (e *DBExecutor) delete(ctx context.Context, table string, where map[string]interface{}) error {
	if where == nil || len(where) == 0 {
		return fmt.Errorf("delete where condition is required")
	}

	whereClauses := make([]string, 0, len(where))
	values := make([]interface{}, 0, len(where))

	for k, v := range where {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", k))
		values = append(values, v)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, strings.Join(whereClauses, " AND "))

	result := e.db.WithContext(ctx).Exec(query, values...)
	if result.Error != nil {
		return fmt.Errorf("delete failed: %w", result.Error)
	}

	logger.Info().
		Str("table", table).
		Int("affected", int(result.RowsAffected)).
		Msg("DB delete completed")

	return nil
}

type ExecutorFactory struct {
	db         *gorm.DB
	mqExecutor *MQExecutor
}

func NewExecutorFactory(db *gorm.DB) (*ExecutorFactory, error) {
	mqExec, err := NewMQExecutor()
	if err != nil {
		return nil, err
	}

	return &ExecutorFactory{
		db:         db,
		mqExecutor: mqExec,
	}, nil
}

func (f *ExecutorFactory) Create(taskType string) (TaskExecutor, error) {
	switch taskType {
	case "rpc":
		return NewRPCExecutor(), nil
	case "mq":
		return f.mqExecutor, nil
	case "http":
		return NewHTTPExecutor(), nil
	case "db":
		return NewDBExecutor(f.db), nil
	default:
		return nil, fmt.Errorf("unsupported task type: %s", taskType)
	}
}
