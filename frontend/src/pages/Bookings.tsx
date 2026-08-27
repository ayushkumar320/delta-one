import { useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { ApiError, api, money, when } from '../api/client'
import type { Booking } from '../api/types'
import { Empty, Loading, Problem } from '../components/Feedback'

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
      <h1>My bookings</h1>
      {justConfirmed && <p className="confirmed">Payment taken. Your tickets are confirmed.</p>}
      {error && <Problem message={error} />}
      {bookings.length === 0 && <Empty message="Nothing booked yet." />}

      <ul className="booking-list">
        {bookings.map((booking) => (
          <li key={booking.id} className={`booking ${booking.status}`}>
            <div>
              <h2>{booking.event_title}</h2>
              <p className="meta">
                {booking.seats.length} seat{booking.seats.length === 1 ? '' : 's'} ·{' '}
                {money(booking.total_cents)} · booked {when(booking.created_at)}
              </p>
            </div>
            <div className="booking-actions">
              <span className={`badge ${booking.status}`}>{booking.status}</span>
              {booking.status === 'held' && (
                <Link className="primary" to={`/checkout/${booking.id}`}>
                  Finish paying
                </Link>
              )}
              {(booking.status === 'held' || booking.status === 'confirmed') && (
                <button type="button" className="ghost" onClick={() => cancel(booking.id)}>
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
