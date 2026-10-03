-- +goose Up
CREATE TABLE refresh_tokens (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id    BIGINT UNSIGNED NOT NULL,
    token_hash CHAR(64)        NOT NULL COMMENT 'sha256 hex digest of the opaque refresh token',
    expires_at DATETIME(6)     NOT NULL,
    revoked_at DATETIME(6)     NULL,
    created_at DATETIME(6)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_refresh_tokens_token_hash (token_hash),
    KEY idx_refresh_tokens_user_id (user_id),
    CONSTRAINT fk_refresh_tokens_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE refresh_tokens;
