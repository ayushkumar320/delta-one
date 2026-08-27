CREATE EXTENSION IF NOT EXISTS citext;

-- citext makes the unique index case-insensitive, so Ada@example.com and
-- ada@example.com cannot both register.
CREATE TABLE users (
    id            uuid PRIMARY KEY,
    email         citext NOT NULL UNIQUE,
    password_hash text NOT NULL,
    name          text NOT NULL,
    role          text NOT NULL DEFAULT 'customer' CHECK (role IN ('customer', 'organizer')),
    created_at    timestamptz NOT NULL DEFAULT now()
);
