<script lang="ts">
  import type { Section } from './workspace';
  let { section, count }: { section: Section; count: number | null } = $props();
  const descriptions: Record<Section, string> = {
    'Unified chat': 'One conversation for your connected agents.',
    Workflows: 'Validate workflow drafts through the local API.',
    Agents: 'Agent connections for this workspace.'
  };
  let rows = $derived(section === 'Workflows'
    ? [['Draft validation', 'Available via API'], ['Visual canvas', 'Planned']]
    : section === 'Agents'
      ? [['Connected agents', count === null ? 'Unknown' : String(count)], ['Provider adapters', 'Planned']]
      : [['Connected agents', count === null ? 'Unknown' : String(count)], ['Task execution', 'Planned']]);
</script>

<section aria-labelledby="section-heading">
  <div class="pattern" aria-hidden="true"><i></i><i></i><i></i></div>
  <h1 id="section-heading">{section}</h1>
  <p>{descriptions[section]}</p>
  <dl>
    {#each rows as [label, value]}
      <div><dt>{label}</dt><dd>{value}</dd></div>
    {/each}
  </dl>
</section>

<style>
  section { margin-block: auto; padding-block: clamp(30px, 7vh, 80px); }
  .pattern { display: flex; align-items: center; height: 58px; color: #a2a2a2; }
  i { width: 33px; height: 33px; border: 1px solid; border-radius: 50%; position: relative; }
  i + i { margin-left: -8px; }
  i:nth-child(2) { width: 47px; height: 47px; color: #ededed; }
  h1 { margin: 12px 0 4px; font-size: clamp(34px, 4vw, 48px); font-weight: 540; letter-spacing: -.055em; line-height: 1.12; }
  p { color: var(--muted); margin: 0; }
  dl { border-top: 1px solid var(--line); margin: 30px 0 0; }
  dl div { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; padding: 10px 0; border-bottom: 1px solid var(--line); }
  dd { margin: 0; color: var(--muted); font: 12px/1.5 ui-monospace, monospace; }
</style>
