import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, money, when } from '../api/client'
import type { Event } from '../api/types'
import { Empty, Loading, Problem } from '../components/Feedback'
import { field } from '../components/ui'

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
      <div className="mb-6 flex flex-wrap items-start justify-between gap-6">
        <div>
          <h1 className="text-3xl font-semibold text-strong">What&rsquo;s on</h1>
          <p className="text-muted">
            Pick your seats, hold them for ten minutes, pay when ready.
          </p>
        </div>
        <input
          type="search"
          value={search}
          placeholder="Search events"
          aria-label="Search events"
          className={`${field} sm:w-64`}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {error && <Problem message={error} />}
      {loading && events.length === 0 && <Loading what="events" />}
      {!loading && events.length === 0 && !error && (
        <Empty message="No events match that search." />
      )}

      <ul className="grid list-none grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-4 p-0">
        {events.map((event) => (
          <li key={event.id}>
            <Link
              to={`/events/${event.id}`}
              className="block h-full rounded-xl border border-edge bg-surface p-5 text-body no-underline hover:border-accent"
            >
              <h2 className="text-xl font-semibold text-strong">{event.title}</h2>
              <p className="mt-1 text-sm text-muted">
                {when(event.starts_at)}
                {event.venue && ` · ${event.venue.name}, ${event.venue.city}`}
              </p>
              <p className="my-3">{event.description}</p>
              <p>
                From <strong className="text-strong">{money(event.from_cents)}</strong>
                <span className="text-muted"> · {event.seat_count} seats</span>
              </p>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
