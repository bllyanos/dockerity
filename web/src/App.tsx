import { BrowserRouter, Navigate, Route, Routes } from 'react-router'
import { AuthProvider } from './auth/AuthProvider'
import { useAuth } from './auth/AuthContext'
import { Layout } from './components/Layout'
import { AppDetailPage } from './pages/apps/AppDetailPage'
import { AppsPage } from './pages/apps/AppsPage'
import { SpecPage } from './pages/apps/SpecPage'
import { AuditPage } from './pages/AuditPage'
import { BuildPage } from './pages/builds/BuildPage'
import { DeploymentPage } from './pages/deployments/DeploymentPage'
import { EnvironmentPage } from './pages/environments/EnvironmentPage'
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
            <Route path="apps" element={<AppsPage />} />
            <Route path="apps/:id" element={<AppDetailPage />} />
            <Route path="apps/:id/spec" element={<SpecPage />} />
            <Route path="builds/:id" element={<BuildPage />} />
            <Route path="environments/:id" element={<EnvironmentPage />} />
            <Route path="deployments/:id" element={<DeploymentPage />} />
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
