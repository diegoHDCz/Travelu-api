-- +goose Up
ALTER TABLE users
    DROP KEY uq_users_username,
    DROP COLUMN username;

-- +goose Down
ALTER TABLE users
    ADD COLUMN username VARCHAR(30) NULL AFTER phone,
    ADD UNIQUE KEY uq_users_username (username);
