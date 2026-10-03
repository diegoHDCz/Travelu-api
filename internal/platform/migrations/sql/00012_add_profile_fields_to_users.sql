-- +goose Up
ALTER TABLE users
    ADD COLUMN first_name VARCHAR(100) NULL AFTER name,
    ADD COLUMN last_name  VARCHAR(100) NULL AFTER first_name,
    ADD COLUMN role       VARCHAR(20)  NOT NULL DEFAULT 'CUSTOMER' COMMENT 'CUSTOMER, VENDOR or ADMIN' AFTER last_name,
    ADD COLUMN is_active  BOOLEAN      NOT NULL DEFAULT TRUE AFTER role;

-- +goose Down
ALTER TABLE users
    DROP COLUMN is_active,
    DROP COLUMN role,
    DROP COLUMN last_name,
    DROP COLUMN first_name;
