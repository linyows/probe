import type { ReactNode } from 'react'

/** `key:` at the start of a line, with whatever indent and value surround it. */
const FIELD = /^(\s*)([A-Za-z_][\w.-]*)(:)(.*)$/

/**
 * A workflow reads by its top-level fields, and by `steps` wherever it sits,
 * so those keys are set in bold and the rest of the line is left alone.
 */
function withFieldsInBold(line: string): ReactNode {
  const match = FIELD.exec(line)
  if (!match) {
    return line
  }

  const [, indent, key, colon, rest] = match
  if (indent !== '' && key !== 'steps') {
    return line
  }

  return (
    <>
      {indent}
      <b className="lp-source__field">
        {key}
        {colon}
      </b>
      {rest}
    </>
  )
}

/** A listing in a framed window, with a gutter of line numbers. */
export function SourcePanel({
  label,
  code,
  className = '',
}: {
  label: string
  code: string
  className?: string
}) {
  return (
    <figure className={`lp-panel lp-panel--paper ${className}`.trim()}>
      <figcaption className="lp-panel__label">{label}</figcaption>
      <pre className="lp-source">
        {/* The lines sit in a box as wide as the longest of them, so the
            banding below reaches the end of a line that scrolls. */}
        <span className="lp-source__lines">
          {code.trimEnd().split('\n').map((line, i) => (
            <span key={i} className="lp-source__line">
              <span className="lp-source__number">{i + 1}</span>
              <span className="lp-source__code">{withFieldsInBold(line) || ' '}</span>
            </span>
          ))}
        </span>
      </pre>
    </figure>
  )
}
