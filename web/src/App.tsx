import { BrowserRouter, Navigate, Route, Routes } from 'react-router'
import { AuthProvider } from './auth/AuthProvider'
import { useAuth } from './auth/AuthContext'
import { Layout } from './components/Layout'
import { AuditPage } from './pages/AuditPage'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { SetupPage } from './pages/SetupPage'
import { TokensPage } from './pages/TokensPage'
import { UsersPage } from './pages/UsersPage'

function Gate() {
  const { state } = useAuth()
  switch (state.status) {
    case 'loading':
      return <p className="p-8 text-sm text-slate-500">Loading…</p>
    case 'setup':
      return <SetupPage />
    case 'anonymous':
      return <LoginPage />
    case 'authenticated':
      return (
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<HomePage />} />
            <Route path="tokens" element={<TokensPage />} />
            {state.user.isAdmin && <Route path="users" element={<UsersPage />} />}
            {state.user.isAdmin && <Route path="audit" element={<AuditPage />} />}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      )
  }
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Gate />
      </AuthProvider>
    </BrowserRouter>
  )
}
