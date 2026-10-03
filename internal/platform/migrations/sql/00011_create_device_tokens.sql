-- +goose Up
CREATE TABLE device_tokens (
    id         CHAR(36)     NOT NULL,
    user_id    CHAR(36)     NOT NULL,
    token      VARCHAR(512) NOT NULL,
    platform   VARCHAR(10)  NOT NULL COMMENT 'ANDROID, IOS or WEB',
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at DATETIME(6)  NOT NULL,
    updated_at DATETIME(6)  NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_device_tokens_token (token),
    KEY idx_device_tokens_user_id (user_id),
    CONSTRAINT fk_device_tokens_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE device_tokens;
