CREATE TABLE venues (
    id         uuid PRIMARY KEY,
    name       text NOT NULL,
    city       text NOT NULL,
    address    text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE events (
    id           uuid PRIMARY KEY,
    venue_id     uuid NOT NULL REFERENCES venues (id) ON DELETE RESTRICT,
    organizer_id uuid,
    title        text NOT NULL,
    description  text NOT NULL DEFAULT '',
    starts_at    timestamptz NOT NULL,
    status       text NOT NULL DEFAULT 'published'
                 CHECK (status IN ('draft', 'published', 'cancelled')),
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- The listing is ordered by start time and filtered to published events.
CREATE INDEX events_published_starts_at_idx
    ON events (starts_at)
    WHERE status = 'published';

-- Seats are the inventory an event sells. Which seats are taken lives in the
-- booking service; catalog only describes what exists and what it costs.
CREATE TABLE seats (
    id          uuid PRIMARY KEY,
    event_id    uuid NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    section     text NOT NULL,
    row_label   text NOT NULL,
    seat_number int  NOT NULL,
    price_cents bigint NOT NULL CHECK (price_cents >= 0),
    UNIQUE (event_id, section, row_label, seat_number)
);

CREATE INDEX seats_event_idx ON seats (event_id);
