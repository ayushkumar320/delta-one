import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ApiError, api, money, when } from '../api/client'
import type { Event, Seat } from '../api/types'
import { Loading, Problem } from '../components/Feedback'
import { useAuth } from '../auth-context'

const MAX_SEATS = 8

export function EventDetail() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()

  const [event, setEvent] = useState<Event | null>(null)
  const [seats, setSeats] = useState<Seat[]>([])
  const [taken, setTaken] = useState<Set<string>>(new Set())
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [holding, setHolding] = useState(false)

  const loadTaken = useCallback(async () => {
    setTaken(new Set(await api.takenSeats(id)))
  }, [id])

  useEffect(() => {
    let cancelled = false
    Promise.all([api.event(id), api.seats(id), api.takenSeats(id)])
      .then(([loadedEvent, loadedSeats, takenIds]) => {
        if (cancelled) return
        setEvent(loadedEvent)
        setSeats(loadedSeats)
        setTaken(new Set(takenIds))
      })
      .catch((err: Error) => !cancelled && setError(err.message))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [id])

  // Someone else's hold can land while this page is open, so the map refreshes
  // rather than letting a visitor pick a seat that is already gone.
  useEffect(() => {
    const timer = setInterval(() => {
      loadTaken().catch(() => undefined)
    }, 15000)
    return () => clearInterval(timer)
  }, [loadTaken])

  const sections = useMemo(() => groupSeats(seats), [seats])
  const total = useMemo(
    () => seats.filter((s) => selected.has(s.id)).reduce((sum, s) => sum + s.price_cents, 0),
    [seats, selected],
  )

  function toggle(seat: Seat) {
    if (taken.has(seat.id)) return
    setSelected((current) => {
      const next = new Set(current)
      if (next.has(seat.id)) next.delete(seat.id)
      else if (next.size < MAX_SEATS) next.add(seat.id)
      return next
    })
  }

  async function hold() {
    if (!user) {
      navigate('/sign-in', { state: { from: `/events/${id}` } })
      return
    }
    setHolding(true)
    setError('')
    try {
      const booking = await api.hold(id, [...selected])
      navigate(`/checkout/${booking.id}`)
    } catch (err) {
      const failure = err as ApiError
      setError(failure.message)
      // Someone won the race for these seats. Refresh the map and clear the
      // selection so the visitor is not staring at seats they cannot have.
      if (failure.code === 'seats_taken') {
        setSelected(new Set())
        await loadTaken().catch(() => undefined)
      }
    } finally {
      setHolding(false)
    }
  }

  if (loading) return <Loading what="the seat map" />
  if (!event) return <Problem message={error || 'That event could not be found.'} />

  return (
    <section className="event-detail">
      <header className="page-head">
        <div>
          <h1>{event.title}</h1>
          <p className="meta">
            {when(event.starts_at)}
            {event.venue && ` · ${event.venue.name}, ${event.venue.city}`}
          </p>
          <p className="blurb">{event.description}</p>
        </div>
      </header>

      {error && <Problem message={error} />}

      <div className="stage">Stage</div>

      <div className="seatmap">
        {sections.map(([section, rows]) => (
          <div key={section} className="section">
            <h3>{section}</h3>
            {rows.map(([row, rowSeats]) => (
              <div key={row} className="row">
                <span className="row-label">{row}</span>
                {rowSeats.map((seat) => {
                  const isTaken = taken.has(seat.id)
                  const isSelected = selected.has(seat.id)
                  return (
                    <button
                      key={seat.id}
                      type="button"
                      className={`seat${isTaken ? ' taken' : ''}${isSelected ? ' selected' : ''}`}
                      disabled={isTaken}
                      aria-pressed={isSelected}
                      title={`${section} ${row}${seat.seat_number} · ${money(seat.price_cents)}${
                        isTaken ? ' · taken' : ''
                      }`}
                      onClick={() => toggle(seat)}
                    >
                      {seat.seat_number}
                    </button>
                  )
                })}
              </div>
            ))}
          </div>
        ))}
      </div>

      <div className="selection-bar">
        <div>
          <strong>{selected.size}</strong> seat{selected.size === 1 ? '' : 's'} selected
          {selected.size > 0 && <span className="muted"> · {money(total)}</span>}
          {selected.size === MAX_SEATS && (
            <span className="muted"> · that&rsquo;s the maximum per booking</span>
          )}
        </div>
        <button
          type="button"
          className="primary"
          disabled={selected.size === 0 || holding}
          onClick={hold}
        >
          {holding ? 'Holding…' : 'Hold these seats'}
        </button>
      </div>
    </section>
  )
}

/** Groups seats into sections, then rows, preserving the API's ordering. */
function groupSeats(seats: Seat[]): [string, [string, Seat[]][]][] {
  const sections = new Map<string, Map<string, Seat[]>>()
  for (const seat of seats) {
    const rows = sections.get(seat.section) ?? new Map<string, Seat[]>()
    rows.set(seat.row_label, [...(rows.get(seat.row_label) ?? []), seat])
    sections.set(seat.section, rows)
  }
  return [...sections].map(([section, rows]) => [section, [...rows]])
}
