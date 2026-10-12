-- +goose Up
-- Baseline Postgres schema. Squashes the earlier MySQL migrations
-- (00001-00013) into their final shape. Constraint names are kept as-is:
-- repositories map unique violations to domain errors by ConstraintName.
-- IDs stay CHAR(36) rather than native UUID on purpose: path IDs reach the
-- repositories unvalidated, and a UUID column would turn a malformed ID into
-- a 22P02 error (500) instead of no rows (404).

CREATE TABLE users (
    id            CHAR(36)     NOT NULL,
    name          VARCHAR(255) NOT NULL,
    first_name    VARCHAR(100) NULL,
    last_name     VARCHAR(100) NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'CUSTOMER', -- CUSTOMER, VENDOR or ADMIN
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    email         VARCHAR(255) NULL,
    phone         VARCHAR(20)  NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL,
    updated_at    TIMESTAMPTZ  NOT NULL,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT uq_users_phone UNIQUE (phone)
);

CREATE TABLE refresh_tokens (
    id         CHAR(36)    NOT NULL,
    user_id    CHAR(36)    NOT NULL,
    token_hash CHAR(64)    NOT NULL, -- sha256 hex digest of the opaque refresh token
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT pk_refresh_tokens PRIMARY KEY (id),
    CONSTRAINT uq_refresh_tokens_token_hash UNIQUE (token_hash),
    CONSTRAINT fk_refresh_tokens_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

CREATE TABLE categories (
    id          CHAR(36)     NOT NULL,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL,
    description TEXT         NULL,
    icon        VARCHAR(255) NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL,
    updated_at  TIMESTAMPTZ  NOT NULL,
    CONSTRAINT pk_categories PRIMARY KEY (id),
    CONSTRAINT uq_categories_name UNIQUE (name),
    CONSTRAINT uq_categories_slug UNIQUE (slug)
);

CREATE TABLE travel_listings (
    id             CHAR(36)      NOT NULL,
    vendor_id      CHAR(36)      NOT NULL,
    title          VARCHAR(255)  NOT NULL,
    description    TEXT          NOT NULL,
    category       VARCHAR(50)   NOT NULL, -- HOTEL, FLIGHT, ACTIVITY or PACKAGE
    location       VARCHAR(255)  NOT NULL,
    city           VARCHAR(100)  NULL,
    country        VARCHAR(100)  NULL,
    price          NUMERIC(10,2) NOT NULL,
    currency       VARCHAR(3)    NOT NULL DEFAULT 'USD',
    capacity       INT           NULL,
    available_from TIMESTAMPTZ   NULL,
    available_to   TIMESTAMPTZ   NULL,
    images         TEXT          NULL, -- JSON array of image URLs
    amenities      TEXT          NULL, -- JSON array
    rating         NUMERIC(3,2)  NOT NULL DEFAULT 0,
    review_count   INT           NOT NULL DEFAULT 0,
    is_active      BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ   NOT NULL,
    updated_at     TIMESTAMPTZ   NOT NULL,
    CONSTRAINT pk_travel_listings PRIMARY KEY (id),
    CONSTRAINT fk_travel_listings_vendor_id FOREIGN KEY (vendor_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX idx_travel_listings_vendor_id ON travel_listings (vendor_id);

CREATE TABLE trip_dates (
    id               CHAR(36)    NOT NULL,
    listing_id       CHAR(36)    NOT NULL,
    start_date       TIMESTAMPTZ NOT NULL,
    end_date         TIMESTAMPTZ NOT NULL,
    max_capacity     INT         NULL, -- overrides the listing capacity when set
    current_bookings INT         NOT NULL DEFAULT 0,
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL,
    CONSTRAINT pk_trip_dates PRIMARY KEY (id),
    CONSTRAINT fk_trip_dates_listing_id FOREIGN KEY (listing_id) REFERENCES travel_listings (id) ON DELETE CASCADE
);
CREATE INDEX idx_trip_dates_listing_id ON trip_dates (listing_id);

CREATE TABLE bookings (
    id               CHAR(36)      NOT NULL,
    customer_id      CHAR(36)      NOT NULL,
    listing_id       CHAR(36)      NOT NULL,
    trip_date_id     CHAR(36)      NULL,
    check_in_date    TIMESTAMPTZ   NULL, -- flexible-date bookings only (backward compatibility)
    check_out_date   TIMESTAMPTZ   NULL, -- flexible-date bookings only (backward compatibility)
    number_of_guests INT           NOT NULL DEFAULT 1,
    total_price      NUMERIC(10,2) NOT NULL,
    currency         VARCHAR(3)    NOT NULL DEFAULT 'USD',
    status           VARCHAR(20)   NOT NULL DEFAULT 'PENDING', -- PENDING, CONFIRMED, CANCELLED or COMPLETED
    payment_status   VARCHAR(20)   NOT NULL DEFAULT 'PENDING', -- PENDING, PAID or REFUNDED
    payment_id       VARCHAR(255)  NULL,
    special_requests TEXT          NULL,
    created_at       TIMESTAMPTZ   NOT NULL,
    updated_at       TIMESTAMPTZ   NOT NULL,
    CONSTRAINT pk_bookings PRIMARY KEY (id),
    CONSTRAINT fk_bookings_customer_id FOREIGN KEY (customer_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_bookings_listing_id FOREIGN KEY (listing_id) REFERENCES travel_listings (id) ON DELETE CASCADE,
    CONSTRAINT fk_bookings_trip_date_id FOREIGN KEY (trip_date_id) REFERENCES trip_dates (id) ON DELETE SET NULL
);
CREATE INDEX idx_bookings_customer_id ON bookings (customer_id);
CREATE INDEX idx_bookings_listing_id ON bookings (listing_id);
CREATE INDEX idx_bookings_trip_date_id ON bookings (trip_date_id);

CREATE TABLE reviews (
    id          CHAR(36)     NOT NULL,
    customer_id CHAR(36)     NOT NULL,
    listing_id  CHAR(36)     NOT NULL,
    booking_id  CHAR(36)     NULL,
    rating      INT          NOT NULL,
    title       VARCHAR(255) NULL,
    comment     TEXT         NULL,
    is_verified BOOLEAN      NOT NULL DEFAULT FALSE, -- verified purchase
    is_approved BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL,
    updated_at  TIMESTAMPTZ  NOT NULL,
    CONSTRAINT pk_reviews PRIMARY KEY (id),
    CONSTRAINT fk_reviews_customer_id FOREIGN KEY (customer_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_reviews_listing_id FOREIGN KEY (listing_id) REFERENCES travel_listings (id) ON DELETE CASCADE,
    CONSTRAINT fk_reviews_booking_id FOREIGN KEY (booking_id) REFERENCES bookings (id) ON DELETE SET NULL,
    CONSTRAINT chk_reviews_rating CHECK (rating BETWEEN 1 AND 5)
);
CREATE INDEX idx_reviews_customer_id ON reviews (customer_id);
CREATE INDEX idx_reviews_listing_id ON reviews (listing_id);
CREATE INDEX idx_reviews_booking_id ON reviews (booking_id);

CREATE TABLE notifications (
    id           CHAR(36)     NOT NULL,
    user_id      CHAR(36)     NOT NULL,
    title        VARCHAR(255) NOT NULL,
    body         TEXT         NOT NULL,
    type         VARCHAR(30)  NOT NULL, -- BOOKING_CONFIRMED, TRIP_REMINDER, RATE_TRIP or BOOKING_CANCELLED
    reference_id CHAR(36)     NULL,
    is_read      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ  NOT NULL,
    CONSTRAINT pk_notifications PRIMARY KEY (id),
    CONSTRAINT fk_notifications_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX idx_notifications_user_id ON notifications (user_id);

CREATE TABLE device_tokens (
    id         CHAR(36)     NOT NULL,
    user_id    CHAR(36)     NOT NULL,
    token      VARCHAR(512) NOT NULL,
    platform   VARCHAR(10)  NOT NULL, -- ANDROID, IOS or WEB
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL,
    CONSTRAINT pk_device_tokens PRIMARY KEY (id),
    CONSTRAINT uq_device_tokens_token UNIQUE (token),
    CONSTRAINT fk_device_tokens_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX idx_device_tokens_user_id ON device_tokens (user_id);

-- +goose Down
DROP TABLE device_tokens;
DROP TABLE notifications;
DROP TABLE reviews;
DROP TABLE bookings;
DROP TABLE trip_dates;
DROP TABLE travel_listings;
DROP TABLE categories;
DROP TABLE refresh_tokens;
DROP TABLE users;
