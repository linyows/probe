import type { MetaRecord } from 'nextra'

export default {
  variables: '変数',
  '-- builtin': { type: 'separator', title: 'ビルトイン' },
  http: 'HTTP',
  smtp: 'SMTP',
  db: 'DB',
  shell: 'SHELL',
  imap: 'IMAP',
  ssh: 'SSH',
  grpc: 'GRPC',
  'mail-latency': 'MAIL-LATENCY',
  hello: 'HELLO',
  embedded: 'EMBEDDED',
  '-- external': { type: 'separator', title: '外部' },
  browser: 'BROWSER',
  graphql: 'GRAPHQL',
  jmap: 'JMAP',
  '-- other': { type: 'separator', title: 'その他' },
  'error-handring': 'エラーハンドリング',
} satisfies MetaRecord
