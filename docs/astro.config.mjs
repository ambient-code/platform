import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import rehypeMermaid from 'rehype-mermaid';

const isNetlify = !!process.env.NETLIFY;

export default defineConfig({
  site: isNetlify
    ? process.env.URL
    : 'https://ambient-code.github.io',
  base: isNetlify ? '/' : '/platform/',
  integrations: [
    starlight({
      title: 'Ambient Code Platform',
      favicon: '/favicon.ico',
      description:
        'AI-powered automation platform for intelligent agentic workflows',
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/ambient-code/platform',
        },
      ],
      editLink: {
        baseUrl:
          'https://github.com/ambient-code/platform/edit/main/docs/',
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { slug: 'getting-started' },
            { slug: 'getting-started/quickstart-ui' },
            { slug: 'getting-started/concepts' },
            { slug: 'getting-started/cli' },
          ],
        },
        {
          label: 'Core Concepts',
          items: [
            { slug: 'concepts/workspaces' },
            { slug: 'concepts/sessions' },
            { slug: 'concepts/integrations' },
            { slug: 'concepts/context-and-artifacts' },
            { slug: 'concepts/workflows' },
            { slug: 'concepts/scheduled-sessions' },
          ],
        },
        {
          label: 'Workflows',
          items: [
            { slug: 'workflows' },
            { slug: 'workflows/bugfix' },
            { slug: 'workflows/triage' },
            { slug: 'workflows/spec-kit' },
            { slug: 'workflows/prd-rfe' },
            { slug: 'workflows/custom' },
          ],
        },
        {
          label: 'Features',
          items: [
            { slug: 'features/session-sharing' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { slug: 'guides/migrating-shared-sessions' },
            { slug: 'guides/continuous-learning-triage' },
          ],
        },
        {
          label: 'Extensions',
          items: [
            { slug: 'extensions/github-action' },
            { slug: 'extensions/mcp-server' },
            { slug: 'extensions/public-api' },
          ],
        },
        {
          label: 'Toolbox',
          items: [
            { slug: 'ecosystem/amber' },
            { slug: 'ecosystem/agentready' },
          ],
        },
        {
          label: 'Development',
          items: [
            { slug: 'development' },
          ],
        },
      ],
      customCss: ['./src/styles/custom.css'],
    }),
  ],
  markdown: {
    rehypePlugins: [[rehypeMermaid, { strategy: 'inline-svg' }]],
  },
});
