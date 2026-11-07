export const sections = ['Unified chat', 'Workflows', 'Agents'] as const;
export type Section = typeof sections[number];
export const symbols: Record<Section, string> = {
  'Unified chat': '⌘',
  Workflows: '◇',
  Agents: '◎'
};
