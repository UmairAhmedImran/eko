import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Eko ✦',
  tagline: 'AI-Powered Snapshot Versioning CLI',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://eko.choreoapps.dev',
  baseUrl: '/',

  organizationName: 'kavix',
  projectName: 'eko',

  onBrokenLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/kavix/eko/tree/main/docs/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/docusaurus-social-card.jpg',
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Eko ✦',
      logo: {
        alt: 'Eko Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'tutorialSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          href: 'https://github.com/kavix/eko',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Getting Started',
              to: '/docs/intro',
            },
            {
              label: 'AI Summaries',
              to: '/docs/ai-summaries',
            },
          ],
        },
        {
          title: 'Community & Source',
          items: [
            {
              label: 'GitHub Repository',
              href: 'https://github.com/kavix/eko',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Eko Project. Built with Docusaurus and deployed on WSO2 Choreo.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'yaml', 'dockerfile', 'diff'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
