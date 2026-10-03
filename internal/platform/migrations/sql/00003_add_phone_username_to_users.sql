-- +goose Up
ALTER TABLE users
    MODIFY COLUMN email VARCHAR(255) NULL,
    ADD COLUMN phone    VARCHAR(20)  NULL AFTER email,
    ADD COLUMN username VARCHAR(30)  NULL AFTER phone,
    ADD UNIQUE KEY uq_users_phone (phone),
    ADD UNIQUE KEY uq_users_username (username);

-- +goose Down
ALTER TABLE users
    DROP KEY uq_users_username,
    DROP KEY uq_users_phone,
    DROP COLUMN username,
    DROP COLUMN phone,
    MODIFY COLUMN email VARCHAR(255) NOT NULL;
