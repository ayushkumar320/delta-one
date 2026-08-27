CREATE TABLE payments (
    id             uuid PRIMARY KEY,
    booking_id     uuid NOT NULL,
    user_id        uuid NOT NULL,
    amount_cents   bigint NOT NULL CHECK (amount_cents >= 0),
    status         text NOT NULL CHECK (status IN ('succeeded', 'failed')),
    failure_reason text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- Idempotency: charging the same booking twice returns the first payment
-- rather than taking the money again.
CREATE UNIQUE INDEX payments_booking_succeeded_idx
    ON payments (booking_id)
    WHERE status = 'succeeded';
