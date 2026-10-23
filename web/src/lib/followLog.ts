export type LogHandlers = {
  /** Called when a (re)connection starts; the server resends the whole log. */
  onReset: () => void
  onChunk: (text: string) => void
  /** Called once the work finished and the log is complete. */
  onEnd: () => void
}

/** Follows a build or deployment log over Server-Sent Events. Returns a function that stops following. */
export function followLog(url: string, { onReset, onChunk, onEnd }: LogHandlers): () => void {
  const source = new EventSource(url)
  source.addEventListener('open', onReset)
  source.addEventListener('log', (e) => onChunk(JSON.parse((e as MessageEvent<string>).data) as string))
  source.addEventListener('end', () => {
    source.close()
    onEnd()
  })
  return () => source.close()
}
