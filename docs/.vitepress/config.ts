import { defineConfig } from 'vitepress'
import { enNavSidebar } from './config-en'
import { jaNavSidebar } from './config-ja'

export default defineConfig({
  title: "Probe",
  description: "A powerful YAML-based workflow automation tool designed for testing, monitoring, and automation tasks. Probe uses plugin-based actions to execute workflows, making it highly flexible and extensible.",
  srcDir: 'src',
  rewrites: {
    'en/:rest*': ':rest*'
  },
  ignoreDeadLinks: true,
  themeConfig: {
    logo: {
      light: '/probe-logo.svg',
      dark: '/probe-logo-dark.svg',
    },
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025-present linyows',
    },
    search: {
      provider: 'local',
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/linyows/probe' }
    ],
    ...enNavSidebar,
  },
  locales: {
    root: {
      label: 'English',
    },
    ja: {
      label: '日本語',
      themeConfig: {
        ...jaNavSidebar,
      },
    },
  },
})
