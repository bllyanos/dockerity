import { useState } from 'react'
import { Button } from './form/Button'

export function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <Button type="button" variant="secondary" onClick={() => void copy()}>
      {copied ? 'Copied' : 'Copy'}
    </Button>
  )
}
