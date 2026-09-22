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
    'A powerful YAML-based workflow automation tool designed for testing, monitoring, and automation tasks. Probe uses plugin-based actions to execute workflows, making it highly flexible and extensible.',
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return <RootHtml lang="en">{children}</RootHtml>
}
