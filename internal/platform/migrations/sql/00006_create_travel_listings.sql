-- +goose Up
CREATE TABLE travel_listings (
    id              CHAR(36)      NOT NULL,
    vendor_id       CHAR(36)      NOT NULL,
    title           VARCHAR(255)  NOT NULL,
    description     TEXT          NOT NULL,
    category        VARCHAR(50)   NOT NULL COMMENT 'HOTEL, FLIGHT, ACTIVITY or PACKAGE',
    location        VARCHAR(255)  NOT NULL,
    city            VARCHAR(100)  NULL,
    country         VARCHAR(100)  NULL,
    price           DECIMAL(10,2) NOT NULL,
    currency        VARCHAR(3)    NOT NULL DEFAULT 'USD',
    capacity        INT           NULL,
    available_from  DATETIME(6)   NULL,
    available_to    DATETIME(6)   NULL,
    images          TEXT          NULL COMMENT 'JSON array of image URLs',
    amenities       TEXT          NULL COMMENT 'JSON array',
    rating          DECIMAL(3,2)  NOT NULL DEFAULT 0,
    review_count    INT           NOT NULL DEFAULT 0,
    is_active       BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at      DATETIME(6)   NOT NULL,
    updated_at      DATETIME(6)   NOT NULL,
    PRIMARY KEY (id),
    KEY idx_travel_listings_vendor_id (vendor_id),
    CONSTRAINT fk_travel_listings_vendor_id FOREIGN KEY (vendor_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE travel_listings;
