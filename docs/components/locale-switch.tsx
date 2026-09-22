'use client'

import Link from 'next/link'
import { useEffect, useId, useRef, useState } from 'react'

type Locale = 'en' | 'ja'

const names = {
  en: 'English',
  ja: '日本語',
} satisfies Record<Locale, string>

const locales = Object.keys(names) as Locale[]

export function LocaleSwitch({
  current,
  hrefs,
}: {
  current: Locale
  hrefs: Record<Locale, string>
}) {
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLDivElement>(null)
  const button = useRef<HTMLButtonElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) {
      return
    }
    const onPointerDown = (event: PointerEvent) => {
      if (!root.current?.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
        button.current?.focus()
      }
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  return (
    <div className="probe-locale" ref={root}>
      <button
        ref={button}
        type="button"
        className="probe-locale__button"
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen(value => !value)}
      >
        {names[current]}
        <span className="probe-locale__caret" aria-hidden="true" />
      </button>
      <ul className="probe-locale__menu" id={menuId} hidden={!open}>
        {locales.map(locale => (
          <li key={locale}>
            <Link
              href={hrefs[locale]}
              className="probe-locale__option"
              aria-current={locale === current ? 'true' : undefined}
              onClick={() => setOpen(false)}
            >
              {names[locale]}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  )
}
