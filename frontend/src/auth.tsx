import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api, storeToken, storedToken } from './api/client'
import type { User } from './api/types'

interface AuthState {
  user: User | null
  loading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signUp: (email: string, password: string, name: string, role: string) => Promise<void>
  signOut: () => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  // Starts true whenever a token is present: the app must not decide the
  // visitor is signed out while that token is still being checked.
  const [loading, setLoading] = useState(storedToken() !== null)

  useEffect(() => {
    if (!storedToken()) return
    let cancelled = false
    api
      .me()
      .then((me) => !cancelled && setUser(me))
      .catch(() => storeToken(null))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [])

  const signIn = useCallback(async (email: string, password: string) => {
    const session = await api.login(email, password)
    storeToken(session.token)
    setUser(session.user)
  }, [])

  const signUp = useCallback(
    async (email: string, password: string, name: string, role: string) => {
      const session = await api.register(email, password, name, role)
      storeToken(session.token)
      setUser(session.user)
    },
    [],
  )

  const signOut = useCallback(() => {
    storeToken(null)
    setUser(null)
  }, [])

  const value = useMemo(
    () => ({ user, loading, signIn, signUp, signOut }),
    [user, loading, signIn, signUp, signOut],
  )
  return <AuthContext value={value}>{children}</AuthContext>
}

export function useAuth(): AuthState {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used inside AuthProvider')
  return context
}
