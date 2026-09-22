import { Noto_Sans_JP, Oswald } from 'next/font/google'

/** Headings. Latin only; tops out at 700. */
export const oswald = Oswald({
  subsets: ['latin'],
  weight: ['400', '500', '700'],
  variable: '--font-display',
  display: 'swap',
})

/** Every Japanese glyph: 400 body, 500 headings, 700 bold runs, 900 the hero. */
export const notoSansJP = Noto_Sans_JP({
  subsets: ['latin'],
  weight: ['400', '500', '700', '900'],
  variable: '--font-jp',
  display: 'swap',
})
