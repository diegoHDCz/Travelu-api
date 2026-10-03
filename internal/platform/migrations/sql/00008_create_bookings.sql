-- +goose Up
CREATE TABLE bookings (
    id               CHAR(36)      NOT NULL,
    customer_id      CHAR(36)      NOT NULL,
    listing_id       CHAR(36)      NOT NULL,
    trip_date_id     CHAR(36)      NULL,
    check_in_date    DATETIME(6)   NULL COMMENT 'flexible-date bookings only (backward compatibility)',
    check_out_date   DATETIME(6)   NULL COMMENT 'flexible-date bookings only (backward compatibility)',
    number_of_guests INT           NOT NULL DEFAULT 1,
    total_price      DECIMAL(10,2) NOT NULL,
    currency         VARCHAR(3)    NOT NULL DEFAULT 'USD',
    status           VARCHAR(20)   NOT NULL DEFAULT 'PENDING' COMMENT 'PENDING, CONFIRMED, CANCELLED or COMPLETED',
    payment_status   VARCHAR(20)   NOT NULL DEFAULT 'PENDING' COMMENT 'PENDING, PAID or REFUNDED',
    payment_id       VARCHAR(255)  NULL,
    special_requests TEXT          NULL,
    created_at       DATETIME(6)   NOT NULL,
    updated_at       DATETIME(6)   NOT NULL,
    PRIMARY KEY (id),
    KEY idx_bookings_customer_id (customer_id),
    KEY idx_bookings_listing_id (listing_id),
    KEY idx_bookings_trip_date_id (trip_date_id),
    CONSTRAINT fk_bookings_customer_id FOREIGN KEY (customer_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_bookings_listing_id FOREIGN KEY (listing_id) REFERENCES travel_listings (id) ON DELETE CASCADE,
    CONSTRAINT fk_bookings_trip_date_id FOREIGN KEY (trip_date_id) REFERENCES trip_dates (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE bookings;
