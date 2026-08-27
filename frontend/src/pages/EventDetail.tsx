import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ApiError, api, money, when } from '../api/client'
import type { Event, Seat } from '../api/types'
import { Loading, Problem } from '../components/Feedback'
import { primaryButton } from '../components/ui'
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
    <section>
      <header className="mb-6">
        <h1 className="text-3xl font-semibold text-strong">{event.title}</h1>
        <p className="mt-1 text-sm text-muted">
          {when(event.starts_at)}
          {event.venue && ` · ${event.venue.name}, ${event.venue.city}`}
        </p>
        <p className="mt-3 max-w-2xl">{event.description}</p>
      </header>

      {error && <Problem message={error} />}

      <div className="mx-auto mt-6 mb-8 max-w-lg rounded-b-[40%] border-t-2 border-accent bg-gradient-to-b from-raised to-transparent p-2 text-center text-xs tracking-[0.3em] text-muted uppercase">
        Stage
      </div>

      <div className="flex flex-col items-center gap-6">
        {sections.map(([section, rows]) => (
          <div key={section}>
            <h3 className="text-center text-xs font-semibold tracking-widest text-muted uppercase">
              {section}
            </h3>
            {rows.map(([row, rowSeats]) => (
              <div key={row} className="mb-1 flex items-center justify-center gap-1.5">
                <span className="w-5 text-xs text-muted">{row}</span>
                {rowSeats.map((seat) => {
                  const isTaken = taken.has(seat.id)
                  const isSelected = selected.has(seat.id)
                  return (
                    <button
                      key={seat.id}
                      type="button"
                      className={seatClass(isTaken, isSelected)}
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

      <div className="sticky bottom-0 mt-8 flex flex-wrap items-center justify-between gap-4 rounded-xl border border-edge bg-surface px-5 py-4">
        <div>
          <strong className="text-strong">{selected.size}</strong> seat
          {selected.size === 1 ? '' : 's'} selected
          {selected.size > 0 && <span className="text-muted"> · {money(total)}</span>}
          {selected.size === MAX_SEATS && (
            <span className="text-muted"> · that&rsquo;s the maximum per booking</span>
          )}
        </div>
        <button
          type="button"
          className={primaryButton}
          disabled={selected.size === 0 || holding}
          onClick={hold}
        >
          {holding ? 'Holding…' : 'Hold these seats'}
        </button>
      </div>
    </section>
  )
}

/** A seat's appearance depends only on whether it is taken or selected. */
function seatClass(isTaken: boolean, isSelected: boolean): string {
  const base = 'h-8 w-8 rounded-lg border text-xs'
  if (isTaken) return `${base} cursor-not-allowed border-dashed border-edge text-edge`
  if (isSelected) return `${base} border-accent bg-accent text-white`
  return `${base} border-edge bg-raised text-body hover:border-accent`
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
