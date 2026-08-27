CREATE TABLE bookings (
    id              uuid PRIMARY KEY,
    user_id         uuid NOT NULL,
    user_email      text NOT NULL,
    event_id        uuid NOT NULL,
    event_title     text NOT NULL,
    status          text NOT NULL
                    CHECK (status IN ('held', 'confirmed', 'cancelled', 'expired')),
    total_cents     bigint NOT NULL CHECK (total_cents >= 0),
    hold_expires_at timestamptz NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX bookings_user_created_idx ON bookings (user_id, created_at DESC);

-- Expiring holds are swept by start time of the scan; this index keeps the
-- sweep from scanning confirmed bookings.
CREATE INDEX bookings_expiring_idx
    ON bookings (hold_expires_at)
    WHERE status = 'held';

CREATE TABLE booking_seats (
    booking_id  uuid NOT NULL REFERENCES bookings (id) ON DELETE CASCADE,
    event_id    uuid NOT NULL,
    seat_id     uuid NOT NULL,
    price_cents bigint NOT NULL CHECK (price_cents >= 0),
    -- Set when the booking is cancelled or its hold expires, which returns the
    -- seat to the pool. An index predicate cannot query another table, so the
    -- claim is tracked here rather than read from bookings.status.
    released_at timestamptz,
    PRIMARY KEY (booking_id, seat_id)
);

-- The double-booking guarantee. A seat can appear in at most one unreleased
-- booking, enforced by the database rather than by application code, so two
-- concurrent requests for the same seat cannot both win.
CREATE UNIQUE INDEX booking_seats_active_seat_idx
    ON booking_seats (event_id, seat_id)
    WHERE released_at IS NULL;

CREATE INDEX booking_seats_event_idx ON booking_seats (event_id) WHERE released_at IS NULL;
