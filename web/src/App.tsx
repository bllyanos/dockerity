import { BrowserRouter, Navigate, Route, Routes } from 'react-router'
import { AuthProvider } from './auth/AuthProvider'
import { useAuth } from './auth/AuthContext'
import { Layout } from './components/Layout'
import { AppBuildsTab } from './pages/apps/AppBuildsTab'
import { AppEnvironmentsTab } from './pages/apps/AppEnvironmentsTab'
import { AppLayout } from './pages/apps/AppLayout'
import { AppRepositoryTab } from './pages/apps/AppRepositoryTab'
import { AppsPage } from './pages/apps/AppsPage'
import { SpecPage } from './pages/apps/SpecPage'
import { AuditPage } from './pages/AuditPage'
import { BuildPage } from './pages/builds/BuildPage'
import { DeploymentPage } from './pages/deployments/DeploymentPage'
import { EnvBackupsTab } from './pages/environments/EnvBackupsTab'
import { EnvConfigTab } from './pages/environments/EnvConfigTab'
import { EnvDeploymentsTab } from './pages/environments/EnvDeploymentsTab'
import { EnvironmentLayout } from './pages/environments/EnvironmentLayout'
import { EnvOverviewTab } from './pages/environments/EnvOverviewTab'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { SetupPage } from './pages/SetupPage'
import { SettingsPage } from './pages/settings/SettingsPage'
import { TokensPage } from './pages/TokensPage'
import { UsersPage } from './pages/UsersPage'

function Gate() {
  const { state } = useAuth()
  switch (state.status) {
    case 'loading':
      return <p className="p-8 body-medium text-on-surface-variant">Loading…</p>
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
            <Route path="apps/:id" element={<AppLayout />}>
              <Route index element={<AppEnvironmentsTab />} />
              <Route path="builds" element={<AppBuildsTab />} />
              <Route path="spec" element={<SpecPage />} />
              <Route path="repository" element={<AppRepositoryTab />} />
            </Route>
            <Route path="builds/:id" element={<BuildPage />} />
            <Route path="environments/:id" element={<EnvironmentLayout />}>
              <Route index element={<EnvOverviewTab />} />
              <Route path="deployments" element={<EnvDeploymentsTab />} />
              <Route path="backups" element={<EnvBackupsTab />} />
              <Route path="config" element={<EnvConfigTab />} />
            </Route>
            <Route path="deployments/:id" element={<DeploymentPage />} />
            <Route path="tokens" element={<TokensPage />} />
            {state.user.isAdmin && <Route path="users" element={<UsersPage />} />}
            {state.user.isAdmin && <Route path="audit" element={<AuditPage />} />}
            {state.user.isAdmin && <Route path="settings" element={<SettingsPage />} />}
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
