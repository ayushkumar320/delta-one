import { Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { RequireAuth } from './components/RequireAuth'
import { Bookings } from './pages/Bookings'
import { Checkout } from './pages/Checkout'
import { EventDetail } from './pages/EventDetail'
import { Events } from './pages/Events'
import { SignIn } from './pages/SignIn'
import { SignUp } from './pages/SignUp'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Events />} />
        <Route path="events/:id" element={<EventDetail />} />
        <Route path="sign-in" element={<SignIn />} />
        <Route path="sign-up" element={<SignUp />} />
        <Route
          path="checkout/:id"
          element={
            <RequireAuth>
              <Checkout />
            </RequireAuth>
          }
        />
        <Route
          path="bookings"
          element={
            <RequireAuth>
              <Bookings />
            </RequireAuth>
          }
        />
        <Route path="*" element={<p className="text-muted">That page does not exist.</p>} />
      </Route>
    </Routes>
  )
}
