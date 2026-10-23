import { HealthStatus } from '../components/HealthStatus'

export function HomePage() {
  return (
    <>
      <h1 className="mb-4 text-xl font-semibold">System status</h1>
      <div className="max-w-md">
        <HealthStatus />
      </div>
    </>
  )
}
