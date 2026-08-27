import type { Booking, Event, Seat, Session, User } from './types'

// Vite proxies /api to the gateway in development, so the same path works in
// both environments.
const BASE = import.meta.env.VITE_API_URL ?? '/api'

const TOKEN_KEY = 'delta-one.token'

export function storedToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function storeToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

/** An error response from the API, carrying the code the services define. */
export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

async function request<T>(
  path: string,
  options: { method?: string; body?: unknown } = {},
): Promise<T> {
  const token = storedToken()
  const response = await fetch(BASE + path, {
    method: options.method ?? 'GET',
    headers: {
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  })

  if (response.status === 204) return undefined as T

  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    // An expired or tampered token is not recoverable; drop it so the app
    // shows a signed-out state instead of retrying with a dead credential.
    if (response.status === 401) storeToken(null)
    throw new ApiError(
      response.status,
      payload?.error?.code ?? 'unknown',
      payload?.error?.message ?? `request failed with ${response.status}`,
    )
  }
  return payload as T
}

export const api = {
  register: (email: string, password: string, name: string, role: string) =>
    request<Session>('/auth/register', {
      method: 'POST',
      body: { email, password, name, role },
    }),

  login: (email: string, password: string) =>
    request<Session>('/auth/login', { method: 'POST', body: { email, password } }),

  me: () => request<User>('/auth/me'),

  events: (search?: string) =>
    request<{ events: Event[] }>(`/events${search ? `?q=${encodeURIComponent(search)}` : ''}`)
      .then((r) => r.events),

  event: (id: string) => request<Event>(`/events/${id}`),

  seats: (eventId: string) =>
    request<{ seats: Seat[] }>(`/events/${eventId}/seats`).then((r) => r.seats),

  takenSeats: (eventId: string) =>
    request<{ seat_ids: string[] }>(`/events/${eventId}/taken-seats`).then((r) => r.seat_ids),

  hold: (eventId: string, seatIds: string[]) =>
    request<Booking>('/bookings', {
      method: 'POST',
      body: { event_id: eventId, seat_ids: seatIds },
    }),

  booking: (id: string) => request<Booking>(`/bookings/${id}`),

  bookings: () => request<{ bookings: Booking[] }>('/bookings').then((r) => r.bookings),

  confirm: (id: string, cardToken: string) =>
    request<Booking>(`/bookings/${id}/confirm`, {
      method: 'POST',
      body: { card_token: cardToken },
    }),

  cancel: (id: string) => request<void>(`/bookings/${id}`, { method: 'DELETE' }),
}

/** Formats cents the way the backend stores them. */
export function money(cents: number): string {
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    maximumFractionDigits: 0,
  }).format(cents / 100)
}

/** Formats an ISO timestamp as a readable date and time. */
export function when(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}
