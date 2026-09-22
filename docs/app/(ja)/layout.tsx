import type { Metadata } from 'next'
import type { ReactNode } from 'react'
import { RootHtml } from '../../components/root-html'
import 'nextra-theme-docs/style.css'
import '../globals.css'
import '../home.css'

export const metadata: Metadata = {
  title: {
    default: 'Probe',
    template: '%s | Probe',
  },
  description:
    'Probeは、実験・テスト・監視・自動化タスクなど、様々な宣言的ワークフローのために設計されたYAMLベースのワークフロー自動化ツールです。',
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return <RootHtml lang="ja">{children}</RootHtml>
}
