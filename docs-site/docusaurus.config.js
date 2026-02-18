// @ts-check
const { themes: prismThemes } = require('prism-react-renderer');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'ZanGuard',
  tagline: 'Zanzibar-inspired fine-grained authorization engine for Go',
  favicon: 'img/favicon.ico',

  url: 'https://zanguard.dev',
  baseUrl: '/',

  organizationName: 'zanguard',
  projectName: 'zanguard',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          routeBasePath: '/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'ZanGuard',
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docs',
            position: 'left',
            label: 'Docs',
          },
          {
            href: 'https://github.com/zanguard/zanguard',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              { label: 'Introduction', to: '/' },
              { label: 'Getting Started', to: '/getting-started' },
              { label: 'Core Concepts', to: '/core-concepts/relation-tuples' },
            ],
          },
          {
            title: 'Reference',
            items: [
              { label: 'Schema DSL', to: '/schema/overview' },
              { label: 'Engine', to: '/engine/check' },
              { label: 'Storage', to: '/storage/overview' },
              { label: 'Multi-Tenancy', to: '/multi-tenancy/overview' },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} ZanGuard. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ['go', 'yaml', 'bash', 'sql'],
      },
      colorMode: {
        defaultMode: 'light',
        disableSwitch: false,
        respectPrefersColorScheme: true,
      },
    }),
};

module.exports = config;
