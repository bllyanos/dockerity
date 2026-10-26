import { Banner } from '../ui/Banner'

export function ErrorMessage({ error }: { error: string | null }) {
  if (!error) return null
  return <Banner tone="error" role="alert">{error}</Banner>
}
