import { useState } from 'react'
import { IconButton } from './ui/IconButton'

export function CopyButton({ text, label = 'Copy' }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return <IconButton icon={copied ? 'check' : 'content_copy'} label={copied ? 'Copied' : label} variant="tonal" onClick={() => void copy()} />
}
