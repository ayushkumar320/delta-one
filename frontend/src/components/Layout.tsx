import { Link, NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth-context'

export function Layout() {
  const { user, signOut } = useAuth()

  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="wordmark">
          Delta<span>One</span>
        </Link>
        <nav>
          <NavLink to="/">Events</NavLink>
          {user && <NavLink to="/bookings">My bookings</NavLink>}
        </nav>
        <div className="account">
          {user ? (
            <>
              <span className="who">{user.name}</span>
              <button type="button" className="ghost" onClick={signOut}>
                Sign out
              </button>
            </>
          ) : (
            <>
              <Link className="ghost" to="/sign-in">
                Sign in
              </Link>
              <Link className="primary" to="/sign-up">
                Create account
              </Link>
            </>
          )}
        </div>
      </header>

      <main>
        <Outlet />
      </main>

      <footer>Delta One — a microservices learning project.</footer>
    </div>
  )
}
