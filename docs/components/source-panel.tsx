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
        {code.trimEnd().split('\n').map((line, i) => (
          <span key={i} className="lp-source__line">
            <span className="lp-source__number">{i + 1}</span>
            <span className="lp-source__code">{line || ' '}</span>
          </span>
        ))}
      </pre>
    </figure>
  )
}
