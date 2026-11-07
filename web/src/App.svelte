<script lang="ts">
  import { onMount } from 'svelte';

  import Sidebar from './lib/Sidebar.svelte';
  import WorkspaceOverview from './lib/WorkspaceOverview.svelte';
  import ChatComposer from './lib/ChatComposer.svelte';
  import type { Section } from './lib/workspace';

  let section = $state<Section>('Unified chat');
  let apiStatus = $state('Checking local API…');
  let agentCount = $state<number | null>(null);

  onMount(() => {
    Promise.all([
      fetch('/healthz').then(r => { if (!r.ok) throw new Error('API unavailable'); return r.json(); }),
      fetch('/api/v1/agents').then(r => { if (!r.ok) throw new Error('API unavailable'); return r.json(); })
    ]).then(([, agents]) => {
      apiStatus = 'Local API connected';
      agentCount = Array.isArray(agents) ? agents.length : 0;
    }).catch(() => {
      apiStatus = 'Go API is not running';
    });
  });
</script>

<svelte:head>
  <meta name="description" content="Confignostic is an open-source control center for independent AI agents." />
</svelte:head>

<div class="shell">
  <Sidebar {section} select={(item) => section = item} status={apiStatus} />
  <main>
    <header>Workspace <span aria-hidden="true">/</span> <strong>{section}</strong></header>
    <div class="content">
      <WorkspaceOverview {section} count={agentCount} />
      {#if section === 'Unified chat'}<ChatComposer />{/if}
    </div>
  </main>
</div>
