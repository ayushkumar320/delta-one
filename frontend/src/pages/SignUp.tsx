import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { ApiError } from '../api/client'
import { useAuth } from '../auth-context'
import { Problem } from '../components/Feedback'

// Matches domain.MinPasswordLength in the auth service, so the browser catches
// a short password before the round trip. The server still enforces it.
const MIN_PASSWORD = 10

export function SignUp() {
  const { user, signUp } = useAuth()
  const navigate = useNavigate()

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('customer')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  if (user) return <Navigate to="/" replace />

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await signUp(email, password, name, role)
      navigate('/', { replace: true })
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="form-page">
      <h1>Create your account</h1>
      {error && <Problem message={error} />}
      <form onSubmit={submit}>
        <label className="field">
          <span>Name</span>
          <input type="text" value={name} required onChange={(e) => setName(e.target.value)} />
        </label>
        <label className="field">
          <span>Email</span>
          <input
            type="email"
            value={email}
            required
            autoComplete="email"
            onChange={(e) => setEmail(e.target.value)}
          />
        </label>
        <label className="field">
          <span>Password</span>
          <input
            type="password"
            value={password}
            required
            minLength={MIN_PASSWORD}
            autoComplete="new-password"
            onChange={(e) => setPassword(e.target.value)}
          />
          <small className="muted">At least {MIN_PASSWORD} characters.</small>
        </label>
        <label className="field">
          <span>I am</span>
          <select value={role} onChange={(e) => setRole(e.target.value)}>
            <option value="customer">Buying tickets</option>
            <option value="organizer">Running events</option>
          </select>
        </label>
        <button type="submit" className="primary" disabled={busy}>
          {busy ? 'Creating…' : 'Create account'}
        </button>
      </form>
      <p className="muted">
        Already have an account? <Link to="/sign-in">Sign in</Link>.
      </p>
    </section>
  )
}
