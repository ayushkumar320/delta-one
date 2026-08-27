import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, money, when } from '../api/client'
import type { Event } from '../api/types'
import { Empty, Loading, Problem } from '../components/Feedback'

export function Events() {
  const [events, setEvents] = useState<Event[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    // Debounced so typing does not fire a request per keystroke.
    const timer = setTimeout(() => {
      setLoading(true)
      api
        .events(search)
        .then((found) => !cancelled && setEvents(found))
        .catch((err: Error) => !cancelled && setError(err.message))
        .finally(() => !cancelled && setLoading(false))
    }, 200)

    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [search])

  return (
    <section>
      <div className="page-head">
        <div>
          <h1>What&rsquo;s on</h1>
          <p className="muted">Pick your seats, hold them for ten minutes, pay when ready.</p>
        </div>
        <input
          type="search"
          value={search}
          placeholder="Search events"
          aria-label="Search events"
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {error && <Problem message={error} />}
      {loading && events.length === 0 && <Loading what="events" />}
      {!loading && events.length === 0 && !error && (
        <Empty message="No events match that search." />
      )}

      <ul className="event-grid">
        {events.map((event) => (
          <li key={event.id}>
            <Link to={`/events/${event.id}`} className="event-card">
              <h2>{event.title}</h2>
              <p className="meta">
                {when(event.starts_at)}
                {event.venue && ` · ${event.venue.name}, ${event.venue.city}`}
              </p>
              <p className="blurb">{event.description}</p>
              <p className="price">
                From <strong>{money(event.from_cents)}</strong>
                <span className="muted"> · {event.seat_count} seats</span>
              </p>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
