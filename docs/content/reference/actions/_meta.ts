import type { MetaRecord } from 'nextra'

export default {
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
} satisfies MetaRecord
