import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ApiError, api, money } from '../api/client'
import type { Booking } from '../api/types'
import { Loading, Problem } from '../components/Feedback'
import { field, ghostButton, primaryButton } from '../components/ui'

// Tokens the simulated gateway recognises, so the failure paths can be tried
// from the UI without a real card.
const CARDS = [
  { token: 'tok_visa', label: 'Card ending 4242 — approves' },
  { token: 'tok_decline', label: 'Card ending 0002 — declined' },
  { token: 'tok_insufficient_funds', label: 'Card ending 9995 — insufficient funds' },
]

export function Checkout() {
  const { id = '' } = useParams()
  const navigate = useNavigate()

  const [booking, setBooking] = useState<Booking | null>(null)
  const [card, setCard] = useState(CARDS[0].token)
  const [error, setError] = useState('')
  const [paying, setPaying] = useState(false)
  const [remaining, setRemaining] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .booking(id)
      .then((found) => !cancelled && setBooking(found))
      .catch((err: Error) => !cancelled && setError(err.message))
    return () => {
      cancelled = true
    }
  }, [id])

  // The hold expires server-side whether or not this tab is open; the
  // countdown only tells the user how long they have left.
  useEffect(() => {
    if (!booking || booking.status !== 'held') return
    const tick = () => {
      const left = new Date(booking.hold_expires_at).getTime() - Date.now()
      setRemaining(Math.max(0, Math.floor(left / 1000)))
    }
    tick()
    const timer = setInterval(tick, 1000)
    return () => clearInterval(timer)
  }, [booking])

  async function pay() {
    setPaying(true)
    setError('')
    try {
      const confirmed = await api.confirm(id, card)
      setBooking(confirmed)
      navigate('/bookings', { state: { confirmed: confirmed.id } })
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setPaying(false)
    }
  }

  async function release() {
    try {
      await api.cancel(id)
      navigate('/')
    } catch (err) {
      setError((err as ApiError).message)
    }
  }

  if (!booking) return error ? <Problem message={error} /> : <Loading what="your hold" />

  const expired = booking.status === 'held' && remaining === 0

  return (
    <section className="mx-auto max-w-lg">
      <h1 className="text-3xl font-semibold text-strong">Checkout</h1>
      <p className="mt-1 text-sm text-muted">
        {booking.seats.length} seat{booking.seats.length === 1 ? '' : 's'} for{' '}
        <strong>{booking.event_title}</strong>
      </p>

      {booking.status === 'held' && remaining !== null && !expired && (
        <p className="mt-4 text-accent">
          Seats held for <strong>{formatCountdown(remaining)}</strong>
        </p>
      )}
      {expired && (
        <Problem message="Your hold expired and the seats went back on sale. Pick them again if they are still free." />
      )}
      {error && <Problem message={error} />}

      <dl className="my-6 rounded-xl border border-edge bg-surface p-4">
        {booking.seats.map((seat) => (
          <div key={seat.seat_id} className="flex justify-between py-1">
            <dt>Seat</dt>
            <dd>{money(seat.price_cents)}</dd>
          </div>
        ))}
        <div className="mt-2 flex justify-between border-t border-edge pt-3 font-semibold text-strong">
          <dt>Total</dt>
          <dd>{money(booking.total_cents)}</dd>
        </div>
      </dl>

      <label className="mb-4 block">
        <span className="mb-1 block text-sm text-muted">Pay with</span>
        <select value={card} className={field} onChange={(e) => setCard(e.target.value)}>
          {CARDS.map((option) => (
            <option key={option.token} value={option.token}>
              {option.label}
            </option>
          ))}
        </select>
      </label>

      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          className={primaryButton}
          disabled={paying || expired || booking.status !== 'held'}
          onClick={pay}
        >
          {paying ? 'Charging…' : `Pay ${money(booking.total_cents)}`}
        </button>
        {booking.status === 'held' && (
          <button type="button" className={ghostButton} onClick={release}>
            Release seats
          </button>
        )}
        {expired && (
          <Link className={ghostButton} to={`/events/${booking.event_id}`}>
            Back to the seat map
          </Link>
        )}
      </div>
    </section>
  )
}

function formatCountdown(seconds: number): string {
  const minutes = Math.floor(seconds / 60)
  return `${minutes}:${String(seconds % 60).padStart(2, '0')}`
}
