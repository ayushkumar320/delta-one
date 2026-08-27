import { useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { ApiError, api, money, when } from '../api/client'
import type { Booking, BookingStatus } from '../api/types'
import { Empty, Loading, Problem } from '../components/Feedback'
import { ghostButton, primaryButton } from '../components/ui'

export function Bookings() {
  const location = useLocation()
  const justConfirmed = (location.state as { confirmed?: string } | null)?.confirmed

  const [bookings, setBookings] = useState<Booking[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    api
      .bookings()
      .then((found) => !cancelled && setBookings(found))
      .catch((err: Error) => !cancelled && setError(err.message))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [])

  async function cancel(id: string) {
    try {
      await api.cancel(id)
      setBookings((current) =>
        current.map((b) => (b.id === id ? { ...b, status: 'cancelled' } : b)),
      )
    } catch (err) {
      setError((err as ApiError).message)
    }
  }

  if (loading) return <Loading what="your bookings" />

  return (
    <section>
      <h1 className="mb-4 text-3xl font-semibold text-strong">My bookings</h1>
      {justConfirmed && (
        <p className="mb-4 rounded-xl border border-good bg-good/10 px-4 py-3 text-emerald-100">
          Payment taken. Your tickets are confirmed.
        </p>
      )}
      {error && <Problem message={error} />}
      {bookings.length === 0 && <Empty message="Nothing booked yet." />}

      <ul className="grid list-none gap-3 p-0">
        {bookings.map((booking) => (
          <li
            key={booking.id}
            className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-edge bg-surface p-4"
          >
            <div>
              <h2 className="text-lg font-semibold text-strong">{booking.event_title}</h2>
              <p className="mt-1 text-sm text-muted">
                {booking.seats.length} seat{booking.seats.length === 1 ? '' : 's'} ·{' '}
                {money(booking.total_cents)} · booked {when(booking.created_at)}
              </p>
            </div>
            <div className="flex items-center gap-3">
              <span className={badge(booking.status)}>{booking.status}</span>
              {booking.status === 'held' && (
                <Link className={primaryButton} to={`/checkout/${booking.id}`}>
                  Finish paying
                </Link>
              )}
              {(booking.status === 'held' || booking.status === 'confirmed') && (
                <button
                  type="button"
                  className={ghostButton}
                  onClick={() => cancel(booking.id)}
                >
                  Cancel
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}

/** Status colours: held is in progress, confirmed is done, the rest are over. */
function badge(status: BookingStatus): string {
  const base = 'rounded-full border px-2 py-0.5 text-xs tracking-wider uppercase'
  switch (status) {
    case 'confirmed':
      return `${base} border-good text-good`
    case 'held':
      return `${base} border-accent text-accent`
    default:
      return `${base} border-bad text-bad`
  }
}
