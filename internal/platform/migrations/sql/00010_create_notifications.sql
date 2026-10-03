-- +goose Up
CREATE TABLE notifications (
    id           CHAR(36)     NOT NULL,
    user_id      CHAR(36)     NOT NULL,
    title        VARCHAR(255) NOT NULL,
    body         TEXT         NOT NULL,
    type         VARCHAR(30)  NOT NULL COMMENT 'BOOKING_CONFIRMED, TRIP_REMINDER, RATE_TRIP or BOOKING_CANCELLED',
    reference_id CHAR(36)     NULL,
    is_read      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at   DATETIME(6)  NOT NULL,
    PRIMARY KEY (id),
    KEY idx_notifications_user_id (user_id),
    CONSTRAINT fk_notifications_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE notifications;
