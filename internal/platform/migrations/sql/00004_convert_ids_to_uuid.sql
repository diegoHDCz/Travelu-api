-- +goose Up
DROP TABLE refresh_tokens;
DROP TABLE users;

CREATE TABLE users (
    id            CHAR(36)     NOT NULL,
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NULL,
    phone         VARCHAR(20)  NULL,
    username      VARCHAR(30)  NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME(6)  NOT NULL,
    updated_at    DATETIME(6)  NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email),
    UNIQUE KEY uq_users_phone (phone),
    UNIQUE KEY uq_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE refresh_tokens (
    id         CHAR(36)    NOT NULL,
    user_id    CHAR(36)    NOT NULL,
    token_hash CHAR(64)    NOT NULL COMMENT 'sha256 hex digest of the opaque refresh token',
    expires_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_refresh_tokens_token_hash (token_hash),
    KEY idx_refresh_tokens_user_id (user_id),
    CONSTRAINT fk_refresh_tokens_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE users;

CREATE TABLE users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name          VARCHAR(255)    NOT NULL,
    email         VARCHAR(255)    NULL,
    phone         VARCHAR(20)     NULL,
    username      VARCHAR(30)     NULL,
    password_hash VARCHAR(255)    NOT NULL,
    created_at    DATETIME(6)     NOT NULL,
    updated_at    DATETIME(6)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email),
    UNIQUE KEY uq_users_phone (phone),
    UNIQUE KEY uq_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

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
