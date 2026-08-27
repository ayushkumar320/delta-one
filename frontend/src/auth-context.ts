import { createContext, useContext } from 'react'
import type { User } from './api/types'

export interface AuthState {
  user: User | null
  loading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signUp: (email: string, password: string, name: string, role: string) => Promise<void>
  signOut: () => void
}

// Kept out of auth.tsx so that file exports only the provider component, which
// is what keeps fast refresh working.
export const AuthContext = createContext<AuthState | null>(null)

export function useAuth(): AuthState {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used inside AuthProvider')
  return context
}
