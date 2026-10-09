import type { MetaRecord } from 'nextra'

export default {
  '-- builtin': { type: 'separator', title: 'Built-in' },
  http: 'HTTP',
  db: 'DB',
  browser: 'BROWSER',
  shell: 'SHELL',
  ssh: 'SSH',
  smtp: 'SMTP',
  imap: 'IMAP',
  grpc: 'GRPC',
  'mail-latency': 'MAIL-LATENCY',
  embedded: 'EMBEDDED',
  hello: 'HELLO',
  '-- external': { type: 'separator', title: 'External' },
  graphql: 'GRAPHQL',
  jmap: 'JMAP',
} satisfies MetaRecord
