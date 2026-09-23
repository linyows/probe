import Link from 'next/link'
import { CopyCommand } from './copy-command'
import { Flowchart } from './flowchart'
import { PluginChart } from './plugin-chart'
import { SourcePanel } from './source-panel'
import { StructureChart } from './structure-chart'
import {
  heroReport,
  heroWorkflow,
  installCommand,
  sharedJob,
  sharedWorkflow,
  sourceCommands,
  type ReportLine,
} from './home-data'
import { homeCopy, type Locale } from './home-copy'

function Report({ lines, animate = false }: { lines: ReportLine[]; animate?: boolean }) {
  return (
    <pre className={`lp-report${animate ? ' lp-report--animate' : ''}`}>
      {lines.map((line, i) => (
        <span
          key={i}
          className={`lp-report__line${line.tone ? ` lp-report__line--${line.tone}` : ''}`}
          style={animate ? { animationDelay: `${120 + i * 90}ms` } : undefined}
        >
          {line.text || ' '}
        </span>
      ))}
    </pre>
  )
}

export function Home({ locale }: { locale: Locale }) {
  const t = homeCopy[locale]

  return (
    <div className="lp">
      <section className="lp-hero">
        <div className="lp-hero__lead">
          <p className="lp-hero__mark">
            <img
              className="lp-hero__markImg lp-hero__markImg--light"
              src="/logo.svg"
              alt=""
            />
            <img
              className="lp-hero__markImg lp-hero__markImg--dark"
              src="/logo-dark.svg"
              alt=""
            />
          </p>
          <h1 className="lp-hero__headline">
            {t.headline.map(line => (
              <span key={line}>{line}</span>
            ))}
          </h1>
          <p className="lp-hero__lede">{t.lede}</p>
          <div className="lp-hero__actions">
            <Link className="lp-button lp-button--solid" href={t.primaryCta.href}>
              {t.primaryCta.label}
            </Link>
            <Link className="lp-button lp-button--quiet" href={t.secondaryCta.href}>
              {t.secondaryCta.label}
            </Link>
          </div>
        </div>

        <h2 className="lp-hero__exampleHeading">{t.exampleHeading}</h2>

        <figure className="lp-hero__flow">
          <Flowchart title={t.heroDagAlt} />
        </figure>

        <p className="lp-hero__flowNote">{t.flowNote}</p>

        <div className="lp-pair">
          <SourcePanel
            label={t.heroWorkflowLabel}
            code={heroWorkflow}
            className="lp-pair__source"
          />

          <figure className="lp-panel lp-panel--readout lp-pair__report">
            <figcaption className="lp-panel__label">{t.heroReportLabel}</figcaption>
            <Report lines={heroReport} animate />
          </figure>
        </div>
      </section>

      <div className="lp-duo">
        <section className="lp-section lp-duo__item">
          <div className="lp-section__head">
            <h2 className="lp-section__heading">{t.graph.heading}</h2>
            <p className="lp-section__body">{t.graph.body}</p>
          </div>
          <figure className="lp-section__chart">
            <StructureChart {...t.graph.chart} />
          </figure>

          <Link className="lp-button lp-button--solid lp-section__link" href={t.graph.link.href}>
            {t.graph.link.label}
          </Link>
        </section>


        <section className="lp-section lp-duo__item">
          <div className="lp-section__head">
            <h2 className="lp-section__heading">{t.channels.heading}</h2>
            <p className="lp-section__body">{t.channels.body}</p>
          </div>
          <figure className="lp-section__chart">
            <PluginChart {...t.channels.chart} />
          </figure>

          <Link className="lp-button lp-button--solid lp-section__link" href={t.channels.link.href}>
            {t.channels.link.label}
          </Link>
        </section>

      </div>

      <section className="lp-section">
        <div className="lp-section__head">
          <h2 className="lp-section__heading">{t.shared.heading}</h2>
          <p className="lp-section__body">{t.shared.body}</p>
        </div>
        <div className="lp-pair lp-pair--top">
          <SourcePanel label={t.shared.workflowLabel} code={sharedWorkflow} />
          <SourcePanel label={t.shared.jobLabel} code={sharedJob} />
        </div>

        <Link className="lp-button lp-button--solid lp-section__link" href={t.shared.link.href}>
          {t.shared.link.label}
        </Link>
      </section>

      <section className="lp-section">
        <div className="lp-section__head">
          <h2 className="lp-section__heading">{t.parts.heading}</h2>
          <p className="lp-section__body">{t.parts.body}</p>
        </div>
        <div className="lp-parts">
          {t.parts.items.map(part => (
            <article key={part.name} className="lp-part">
              <h3 className="lp-part__name">{part.name}</h3>
              <p className="lp-part__description">{part.description}</p>
              <SourcePanel label="yaml" code={part.code} className="lp-part__code" />
            </article>
          ))}
        </div>
      </section>

      <section className="lp-section lp-section--install">
        <div className="lp-section__head">
          <h2 className="lp-section__heading">{t.install.heading}</h2>
          <p className="lp-section__body">{t.install.body}</p>
        </div>
        <div className="lp-install">
          <CopyCommand command={installCommand} labels={t.copy} />
          <div className="lp-install__source">
            <h3 className="lp-install__sourceLabel">{t.install.sourceLabel}</h3>
            <pre className="lp-install__code">{sourceCommands}</pre>
          </div>
          <Link className="lp-button lp-button--solid lp-install__link" href={t.install.docsHref}>
            {t.install.docsLink}
          </Link>
        </div>
      </section>
    </div>
  )
}
