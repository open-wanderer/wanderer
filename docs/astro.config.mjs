import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import node from "@astrojs/node";

import starlightOpenAPI, { openAPISidebarGroups } from 'starlight-openapi'
import tailwindcss from "@tailwindcss/vite";

import svelte from '@astrojs/svelte';

// https://astro.build/config
export default defineConfig({
  integrations: [starlight({
    title: 'wanderer Documentation',
    logo: {
      light: '/src/assets/logo_text_dark.svg',
      dark: '/src/assets/logo_text_light.svg',
      replacesTitle: true
    },
    social: [
      { icon: 'github', label: 'GitHub', href: 'https://github.com/open-wanderer/wanderer' },
    ],
    components: {
      Footer: './src/components/footer.astro'
    },
    plugins: [
      starlightOpenAPI([
        {
          base: 'api-reference',
          label: 'API Reference',
          schema: 'wanderer.openapi.json',
          sidebar: {
            operations: {
              
            }
          }
        },
      ]),
    ],
    sidebar: [
      {
        label: 'Welcome to wanderer',
        link: '/welcome'
      },
      {
        label: 'Using wanderer',
        items: [{
          label: 'Authentication',
          link: '/use/authentication/'
        }, {
          label: 'Create/Edit a trail',
          link: '/use/create-a-trail/'
        },
        {
          label: 'Categories',
          link: '/use/categories/'
        },
        {
          label: 'Summit logs',
          link: '/use/summit-logs/'
        },
        {
          label: 'Merge trails',
          link: '/use/merge-trails/'
        },
        {
          label: 'Interact with the community',
          link: '/use/community-interaction/'
        },
        {
          label: 'Share trails',
          link: '/use/share-trails/'
        }, {
          label: 'Lists',
          link: '/use/lists/'
        },
        {
          label: 'Statistics',
          link: '/use/statistics/'
        },
        {
          label: 'Customize the map',
          link: '/use/customize-map/'
        },
        {
          label: 'Import/Export',
          link: '/use/import-export/'
        },
        {
          label: 'Plugins',
          link: '/use/plugins/'
        },
        ]
      },
      {
        label: 'Running wanderer',
        items: [
          {
            label: 'Installation',
            items: [
              { label: 'Quickstart', link: '/run/installation/quick' },
              { label: 'Manual Docker Setup', link: '/run/installation/docker' },
              { label: 'Install from Source', link: '/run/installation/from-source' },
              { label: 'Plugin installation', link: '/run/installation/plugins' },
            ]
          },
          {
            label: 'Environment configuration',
            link: '/run/environment-configuration/'
          },
          {
            label: 'Frontend configuration',
            items: [
              { label: 'Edit the "About" section', link: '/run/frontend-configuration/about' },
            ]
          },
          {
            label: 'Backend configuration',
            items: [
              { label: 'Overview', link: '/run/backend-configuration/' },
              { label: 'SMTP', link: '/run/backend-configuration/smtp/' },
              { label: 'Authentication Providers', link: '/run/backend-configuration/auth-providers/' },
              { label: 'Backing up your server', link: '/run/backend-configuration/backup-server/' },
              { label: 'Custom categories', link: '/run/backend-configuration/custom-categories/' },
              { label: 'Adjust Filesize Limits', link: '/run/backend-configuration/adjust-filesize-limits/' },
            ]
          }
        ]
      },
      {
        label: 'Develop wanderer',
        items: [
          {
            label: 'Local development',
            link: '/develop/local-development/'
          },
          {
            label: 'API',
            link: '/develop/api/'
          },
          {
            label: 'Federation',
            link: '/develop/federation/'
          },
          {
            label: 'Plugin System',
            link: '/develop/plugin-system/'
          },
          {
            label: 'Design-Spezifikationen',
            collapsed: true,
            items: [
              {
                label: 'Überblick',
                link: '/develop/specs/trail-search/'
              },
              {
                label: 'Gemeinsame Invarianten',
                link: '/develop/specs/trail-search/shared-invariants/'
              },
              {
                label: 'Delivery und Beiträge',
                link: '/develop/specs/trail-search/delivery/'
              },
              {
                label: 'Capabilities',
                collapsed: true,
                items: [{
                  autogenerate: {
                    directory: 'develop/specs/trail-search/capabilities',
                    collapsed: true,
                  }
                }]
              },
              {
                label: 'Work Items',
                collapsed: true,
                items: [
                  {
                    label: 'Übersicht',
                    link: '/develop/specs/trail-search/work-items/'
                  },
                  {
                    label: 'Suche und Panel',
                    collapsed: true,
                    items: [{
                      autogenerate: {
                        directory: 'develop/specs/trail-search/work-items/search',
                        collapsed: true,
                      }
                    }]
                  },
                  {
                    label: 'Engine',
                    collapsed: true,
                    items: [{
                      autogenerate: {
                        directory: 'develop/specs/trail-search/work-items/engine',
                        collapsed: true,
                      }
                    }]
                  },
                  {
                    label: 'Geo-Discovery',
                    collapsed: true,
                    items: [{
                      autogenerate: {
                        directory: 'develop/specs/trail-search/work-items/geo-discovery',
                        collapsed: true,
                      }
                    }]
                  },
                  {
                    label: 'Typisierte Ortssuche',
                    collapsed: true,
                    items: [{
                      autogenerate: {
                        directory: 'develop/specs/trail-search/work-items/place-search',
                        collapsed: true,
                      }
                    }]
                  },
                  {
                    label: 'Counts und Histogramme',
                    collapsed: true,
                    items: [{
                      autogenerate: {
                        directory: 'develop/specs/trail-search/work-items/aggregations',
                        collapsed: true,
                      }
                    }]
                  }
                ]
              },
              {
                label: 'Verträge',
                collapsed: true,
                items: [{
                  autogenerate: {
                    directory: 'develop/specs/trail-search/contracts',
                    collapsed: true,
                  }
                }]
              },
              {
                label: 'Entscheidungen',
                collapsed: true,
                items: [{
                  autogenerate: {
                    directory: 'develop/specs/trail-search/decisions',
                    collapsed: true,
                  }
                }]
              },
              {
                label: 'Evidenz und Kalibrierung',
                collapsed: true,
                items: [{
                  autogenerate: {
                    directory: 'develop/specs/trail-search/evidence',
                    collapsed: true,
                  }
                }]
              }
            ]
          },
        ]
      },
      ...openAPISidebarGroups,
      {
        label: 'Changelog',
        link: '/changelog/',
      }],
    customCss: ['./src/custom.css', './src/tailwind.css', '@fontsource/ibm-plex-sans/400.css', '@fontsource/ibm-plex-sans/600.css', '@fontsource/ibm-plex-mono/400.css', '@fontsource/ibm-plex-mono/600.css']
  }), svelte()],
  output: "server",
  vite: { plugins: [tailwindcss()] },
  adapter: node({
    mode: "standalone"
  }),
  redirects: {
    '/run/changelog/': '/changelog/',
    '/run/backup-server/': '/run/backend-configuration/backup-server/',
    '/run/custom-categories/': '/run/backend-configuration/custom-categories/',
  },
});
