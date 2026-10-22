import { HealthStatus } from './components/HealthStatus'
import { Layout } from './components/Layout'

export default function App() {
  return (
    <Layout>
      <h1 className="mb-4 text-xl font-semibold">System status</h1>
      <div className="max-w-md">
        <HealthStatus />
      </div>
    </Layout>
  )
}
