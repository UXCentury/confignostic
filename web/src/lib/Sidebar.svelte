<script lang="ts">
  import { sections, symbols, type Section } from './workspace';
  let { section, select, status }: {
    section: Section;
    select: (item: Section) => void;
    status: string;
  } = $props();
</script>

<aside>
  <div class="brand"><span aria-hidden="true">◎</span>Confignostic</div>
  <nav aria-label="Workspace">
    {#each sections as item}
      <button aria-current={section === item ? 'page' : undefined} onclick={() => select(item)}>
        <span aria-hidden="true">{symbols[item]}</span>{item}
      </button>
    {/each}
  </nav>
  <div class="status" role="status">{status}</div>
</aside>

<style>
  aside { background: var(--rail); border-right: 1px solid var(--line); padding: 0 12px; display: flex; flex-direction: column; }
  .brand { height: 62px; display: flex; align-items: center; gap: 9px; padding-inline: 8px; font-size: 17px; font-weight: 620; letter-spacing: -.045em; }
  .brand span { font-size: 26px; }
  nav { display: grid; gap: 3px; margin-top: 18px; }
  button { background: transparent; border: 0; border-radius: 5px; display: flex; gap: 12px; align-items: center; padding: 9px 12px; color: var(--muted); cursor: pointer; }
  button:hover, button[aria-current] { background: var(--surface); color: var(--text); }
  button span { width: 16px; font-size: 16px; }
  .status { margin-top: auto; border-top: 1px solid var(--line); padding: 13px 8px; color: var(--muted); font-size: 12px; }
  @media (max-width: 680px) {
    aside { display: grid; grid-template-columns: auto 1fr; border: 0; border-bottom: 1px solid var(--line); }
    .brand { height: 48px; }
    .status { grid-area: 1 / 2; align-self: center; text-align: right; margin: 0; padding: 0; border: 0; }
    nav { grid-column: 1 / -1; grid-template-columns: repeat(3, 1fr); margin: 0 0 8px; }
    button { justify-content: center; padding: 6px 2px; gap: 5px; }
  }
</style>
