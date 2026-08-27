export type Role = 'customer' | 'organizer'

export interface User {
  id: string
  email: string
  name: string
  role: Role
  created_at: string
}

export interface Session {
  token: string
  expires_at: string
  user: User
}

export interface Venue {
  id: string
  name: string
  city: string
  address: string
}

export interface Event {
  id: string
  venue_id: string
  title: string
  description: string
  starts_at: string
  status: 'draft' | 'published' | 'cancelled'
  venue?: Venue
  seat_count: number
  from_cents: number
}

export interface Seat {
  id: string
  event_id: string
  section: string
  row_label: string
  seat_number: number
  price_cents: number
}

export interface BookingSeat {
  seat_id: string
  price_cents: number
}

export type BookingStatus = 'held' | 'confirmed' | 'cancelled' | 'expired'

export interface Booking {
  id: string
  user_id: string
  event_id: string
  event_title: string
  status: BookingStatus
  total_cents: number
  hold_expires_at: string
  created_at: string
  seats: BookingSeat[]
}
