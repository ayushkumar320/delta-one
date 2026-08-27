import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { ApiError } from '../api/client'
import { useAuth } from '../auth-context'
import { Problem } from '../components/Feedback'
import { field, primaryButton } from '../components/ui'

export function SignIn() {
  const { user, signIn } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? '/'

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  if (user) return <Navigate to={from} replace />

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await signIn(email, password)
      navigate(from, { replace: true })
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="mx-auto max-w-md">
      <h1 className="mb-4 text-3xl font-semibold text-strong">Sign in</h1>
      {error && <Problem message={error} />}
      <form onSubmit={submit}>
        <label className="mb-4 block">
          <span className="mb-1 block text-sm text-muted">Email</span>
          <input
            type="email"
            value={email}
            required
            autoComplete="email"
            className={field}
            onChange={(e) => setEmail(e.target.value)}
          />
        </label>
        <label className="mb-4 block">
          <span className="mb-1 block text-sm text-muted">Password</span>
          <input
            type="password"
            value={password}
            required
            autoComplete="current-password"
            className={field}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        <button type="submit" className={primaryButton} disabled={busy}>
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
      <p className="mt-4 text-muted">
        No account yet? <Link to="/sign-up">Create one</Link>.
      </p>
    </section>
  )
}
