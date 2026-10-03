-- +goose Up
CREATE TABLE trip_dates (
    id               CHAR(36)    NOT NULL,
    listing_id       CHAR(36)    NOT NULL,
    start_date       DATETIME(6) NOT NULL,
    end_date         DATETIME(6) NOT NULL,
    max_capacity     INT         NULL COMMENT 'overrides the listing capacity when set',
    current_bookings INT         NOT NULL DEFAULT 0,
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       DATETIME(6) NOT NULL,
    updated_at       DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_trip_dates_listing_id (listing_id),
    CONSTRAINT fk_trip_dates_listing_id FOREIGN KEY (listing_id) REFERENCES travel_listings (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE trip_dates;
