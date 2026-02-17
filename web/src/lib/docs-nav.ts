export interface NavItem {
  label: string
  to: string
  icon: string
}

export interface NavSection {
  title: string
  items: NavItem[]
}

export const docsNav: NavSection[] = [
  {
    title: 'Getting Started',
    items: [
      { label: 'Quickstart', to: '/docs/quickstart', icon: '$' },
      { label: 'Daemon Setup', to: '/docs/daemon', icon: '>' },
      { label: 'MCP Server', to: '/docs/mcp', icon: '>>>' },
      { label: 'Local Setup', to: '/docs/local-setup', icon: '#' },
    ],
  },
  {
    title: 'Concepts',
    items: [
      { label: 'Architecture', to: '/docs/architecture', icon: '[~]' },
      { label: 'Sandboxes', to: '/docs/sandboxes', icon: 'ls' },
      { label: 'Upgrade to Hosted', to: '/docs/upgrade', icon: '/' },
    ],
  },
  {
    title: 'Reference',
    items: [
      { label: 'API Reference', to: '/docs/api', icon: '{}' },
      { label: 'TUI & MCP Reference', to: '/docs/cli-reference', icon: '$_' },
    ],
  },
]

export const flatNav: NavItem[] = docsNav.flatMap((s) => s.items)
