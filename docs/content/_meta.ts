import type { MetaRecord } from 'nextra'

export default {
  index: {
    type: 'page',
    display: 'hidden',
    theme: {
      layout: 'full',
      sidebar: false,
      toc: false,
      breadcrumb: false,
      pagination: false,
      timestamp: false,
    },
  },
  guide: {
    type: 'page',
    title: 'Guide',
  },
  reference: {
    type: 'page',
    title: 'Reference',
  },
  external: {
    type: 'menu',
    title: 'External',
    items: {
      release: {
        title: 'Github Release',
        href: 'https://github.com/linyows/probe/releases',
      },
      godoc: {
        title: 'Go Docs',
        href: 'https://pkg.go.dev/github.com/linyows/probe',
      },
      deepwiki: {
        title: 'Deep Wiki',
        href: 'https://deepwiki.com/linyows/probe',
      },
    },
  },
  ja: {
    type: 'page',
    display: 'hidden',
  },
} satisfies MetaRecord
