-- +goose Up
-- +goose StatementBegin

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

CREATE TABLE task_group_instance (
    id VARCHAR(64) PRIMARY KEY,
    flow_id VARCHAR(64) NOT NULL,
    status ENUM('pending', 'running', 'success', 'failed') DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    INDEX idx_flow_status (flow_id, status)
);

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
    INDEX idx_group_id (group_id),
    INDEX idx_retry_strategy (retry_strategy, handled)
);

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

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS execution_log;
DROP TABLE IF EXISTS exception_record;
DROP TABLE IF EXISTS dist_task;
DROP TABLE IF EXISTS task_group_instance;
DROP TABLE IF EXISTS task_group_flow;

-- +goose StatementEnd
