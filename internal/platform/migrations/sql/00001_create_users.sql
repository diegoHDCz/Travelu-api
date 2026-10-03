-- +goose Up
CREATE TABLE users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name          VARCHAR(255)    NOT NULL,
    email         VARCHAR(255)    NOT NULL,
    password_hash VARCHAR(255)    NOT NULL,
    created_at    DATETIME(6)     NOT NULL,
    updated_at    DATETIME(6)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE users;
