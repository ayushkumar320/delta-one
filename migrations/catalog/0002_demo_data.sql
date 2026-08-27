-- Demo data, so a fresh checkout has something to browse. Start times are
-- relative to when the migration runs, which keeps the seeded events in the
-- future and therefore visible in the listing.
INSERT INTO venues (id, name, city, address) VALUES
    ('11111111-1111-4111-8111-111111111111', 'Aurora Hall',   'Bengaluru', '12 Residency Road'),
    ('22222222-2222-4222-8222-222222222222', 'Pier Nine',     'Mumbai',    '9 Marine Drive'),
    ('33333333-3333-4333-8333-333333333333', 'The Glasshouse','Delhi',     '44 Hauz Khas');

INSERT INTO events (id, venue_id, title, description, starts_at, status) VALUES
    ('aaaaaaaa-0000-4000-8000-000000000001',
     '11111111-1111-4111-8111-111111111111',
     'Midnight Synth Orchestra',
     'A seventeen-piece ensemble playing analogue synth arrangements of film scores.',
     now() + interval '14 days', 'published'),
    ('aaaaaaaa-0000-4000-8000-000000000002',
     '22222222-2222-4222-8222-222222222222',
     'Harbour Jazz Sessions',
     'Three sets, one stage, and a view of the water that does most of the work.',
     now() + interval '21 days', 'published'),
    ('aaaaaaaa-0000-4000-8000-000000000003',
     '33333333-3333-4333-8333-333333333333',
     'Longform: An Evening of Improv',
     'No script, no safety net, one suggestion from the audience.',
     now() + interval '35 days', 'published');

-- Three sections per event, priced by proximity to the stage.
INSERT INTO seats (id, event_id, section, row_label, seat_number, price_cents)
SELECT
    gen_random_uuid(),
    e.id,
    s.section,
    r.row_label,
    n.seat_number,
    s.price_cents
FROM events e
CROSS JOIN (VALUES
    ('Front',  450000),
    ('Middle', 280000),
    ('Rear',   150000)
) AS s (section, price_cents)
CROSS JOIN (VALUES ('A'), ('B'), ('C')) AS r (row_label)
CROSS JOIN generate_series(1, 8) AS n (seat_number)
WHERE e.id IN (
    'aaaaaaaa-0000-4000-8000-000000000001',
    'aaaaaaaa-0000-4000-8000-000000000002',
    'aaaaaaaa-0000-4000-8000-000000000003'
);
