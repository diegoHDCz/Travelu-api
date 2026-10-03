-- +goose Up
CREATE TABLE reviews (
    id          CHAR(36)     NOT NULL,
    customer_id CHAR(36)     NOT NULL,
    listing_id  CHAR(36)     NOT NULL,
    booking_id  CHAR(36)     NULL,
    rating      INT          NOT NULL,
    title       VARCHAR(255) NULL,
    comment     TEXT         NULL,
    is_verified BOOLEAN      NOT NULL DEFAULT FALSE COMMENT 'verified purchase',
    is_approved BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  DATETIME(6)  NOT NULL,
    updated_at  DATETIME(6)  NOT NULL,
    PRIMARY KEY (id),
    KEY idx_reviews_customer_id (customer_id),
    KEY idx_reviews_listing_id (listing_id),
    KEY idx_reviews_booking_id (booking_id),
    CONSTRAINT fk_reviews_customer_id FOREIGN KEY (customer_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_reviews_listing_id FOREIGN KEY (listing_id) REFERENCES travel_listings (id) ON DELETE CASCADE,
    CONSTRAINT fk_reviews_booking_id FOREIGN KEY (booking_id) REFERENCES bookings (id) ON DELETE SET NULL,
    CONSTRAINT chk_reviews_rating CHECK (rating BETWEEN 1 AND 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE reviews;
