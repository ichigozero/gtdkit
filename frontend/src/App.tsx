import * as React from 'react'
import {
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from 'react-router-dom'

import { authProvider } from './api/auth/auth'
import AuthContext from './context/AuthContext'
import useAuth from './hooks/useAuth'
import { Tokens } from './types'
import HomePage from './views/Home/HomePage'
import LoginPage from './views/Login/LoginPage'

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <RequireAuth>
                <HomePage />
              </RequireAuth>
            }
          />
        </Route>
      </Routes>
    </AuthProvider>
  )
}

function AuthProvider({ children }: { children: React.ReactNode }) {
  const [tokens, setTokens] = React.useState<Tokens | null>(null)

  const signin = (
    username: string,
    password: string,
    callback: (error: any) => void
  ) => {
    return authProvider.signin(
      username,
      password,
      (error: any, tokens: Tokens | null) => {
        tokens && setTokens(tokens)
        callback(error)
      }
    )
  }

  const signout = (callback: (error: any) => void) => {
    if (!tokens) return

    return authProvider.signout(tokens.access, (error: any) => {
      setTokens(null)
      callback(error)
    })
  }

  const value = { tokens, signin, signout }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

function Layout() {
  return (
    <div>
      <AuthStatus />
      <Outlet />
    </div>
  )
}

function AuthStatus() {
  const auth = useAuth()
  const navigate = useNavigate()

  if (!auth.tokens) {
    return null
  }

  return (
    <p>
      <button
        onClick={() => {
          auth.signout(() => navigate('/'))
        }}
      >
        Sign out
      </button>
    </p>
  )
}

function RequireAuth({ children }: { children: JSX.Element }) {
  const auth = useAuth()
  const location = useLocation()

  if (!auth.tokens) {
    // Redirect them to the /login page, but save the current location they were
    // trying to go to when they were redirected. This allows us to send them
    // along to that page after they login, which is a nicer user experience
    // than dropping them off on the home page.
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return children
}
