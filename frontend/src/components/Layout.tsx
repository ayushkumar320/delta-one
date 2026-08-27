import { Link, NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth-context'
import { ghostButton, primaryButton } from './ui'

const navLink = ({ isActive }: { isActive: boolean }) =>
  `no-underline py-1 ${isActive ? 'text-strong' : 'text-muted hover:text-strong'}`

export function Layout() {
  const { user, signOut } = useAuth()

  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex items-center gap-6 border-b border-edge bg-surface px-5 py-4 sm:px-8">
        <Link to="/" className="text-lg font-bold text-strong no-underline">
          Delta<span className="text-accent">One</span>
        </Link>
        <nav className="flex flex-1 gap-4">
          <NavLink to="/" end className={navLink}>
            Events
          </NavLink>
          {user && (
            <NavLink to="/bookings" className={navLink}>
              My bookings
            </NavLink>
          )}
        </nav>
        <div className="flex items-center gap-3">
          {user ? (
            <>
              <span className="hidden text-muted sm:inline">{user.name}</span>
              <button type="button" className={ghostButton} onClick={signOut}>
                Sign out
              </button>
            </>
          ) : (
            <>
              <Link className={ghostButton} to="/sign-in">
                Sign in
              </Link>
              <Link className={primaryButton} to="/sign-up">
                Create account
              </Link>
            </>
          )}
        </div>
      </header>

      <main className="mx-auto my-8 w-full max-w-5xl flex-1 px-5">
        <Outlet />
      </main>

      <footer className="border-t border-edge p-6 text-center text-muted">
        Delta One — a microservices learning project.
      </footer>
    </div>
  )
}
