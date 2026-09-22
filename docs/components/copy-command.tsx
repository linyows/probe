'use client'

import { useState } from 'react'

export function CopyCommand({
  command,
  labels,
}: {
  command: string
  labels: { idle: string; done: string }
}) {
  const [copied, setCopied] = useState(false)

  return (
    <div className="lp-command">
      <code className="lp-command__text">
        <span className="lp-command__prompt">$</span> {command}
      </code>
      <button
        type="button"
        className="lp-command__button"
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(command)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          } catch {
            setCopied(false)
          }
        }}
      >
        {copied ? labels.done : labels.idle}
      </button>
    </div>
  )
}
