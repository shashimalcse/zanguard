// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    {
      type: 'doc',
      id: 'intro',
      label: 'Introduction',
    },
    {
      type: 'doc',
      id: 'getting-started',
      label: 'Getting Started',
    },
    {
      type: 'category',
      label: 'Core Concepts',
      collapsed: false,
      items: [
        'core-concepts/relation-tuples',
        'core-concepts/permissions',
        'core-concepts/abac-conditions',
      ],
    },
    {
      type: 'category',
      label: 'Schema DSL',
      collapsed: false,
      items: [
        'schema/overview',
        'schema/types-and-relations',
        'schema/permissions',
        'schema/conditions',
      ],
    },
    {
      type: 'category',
      label: 'Engine',
      collapsed: false,
      items: [
        'engine/check',
        'engine/expand',
        'engine/cycle-detection',
      ],
    },
    {
      type: 'category',
      label: 'Storage',
      collapsed: false,
      items: [
        'storage/overview',
        'storage/postgresql',
        'storage/changelog',
      ],
    },
    {
      type: 'category',
      label: 'Multi-Tenancy',
      collapsed: false,
      items: [
        'multi-tenancy/overview',
        'multi-tenancy/lifecycle',
        'multi-tenancy/schema-modes',
        'multi-tenancy/context',
      ],
    },
    {
      type: 'category',
      label: 'Examples',
      collapsed: false,
      items: [
        'examples/google-drive',
        'examples/group-membership',
        'examples/abac-clearance',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      collapsed: false,
      items: [
        'api/overview',
        'api/management',
        'api/authzen',
      ],
    },
  ],
};

module.exports = sidebars;
