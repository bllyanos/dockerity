import { useState } from 'react'
import { updateEnvironmentDomain, type Environment } from '../../api/environments'
import { ErrorMessage } from '../../components/form/ErrorMessage'
import { useSubmit } from '../../components/form/useSubmit'
import { Button } from '../../components/ui/Button'
import { Section } from '../../components/ui/Section'
import { TextField } from '../../components/ui/TextField'

/** The host Traefik routes to this environment. Ephemeral environments get theirs from the base domain. */
export function DomainPanel({ env, onChanged }: { env: Environment; onChanged: (e: Environment) => void }) {
  const [domain, setDomain] = useState(env.domain)
  const save = useSubmit(async () => {
    onChanged(await updateEnvironmentDomain(env.id, domain))
  })

  return (
    <Section title="Domain" description="Traefik routes this host to the environment. Empty means not routed.">
      <form onSubmit={(e) => void save.onSubmit(e)} className="flex flex-col gap-4">
        <TextField label="Domain" name="domain" leadingIcon="public" value={domain} onChange={(e) => setDomain(e.target.value)} />
        <ErrorMessage error={save.error} />
        <div className="flex justify-end">
          <Button type="submit" variant="tonal" disabled={save.pending || domain === env.domain}>Save domain</Button>
        </div>
      </form>
    </Section>
  )
}
