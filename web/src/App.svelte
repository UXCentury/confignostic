<script lang="ts">
  import { onMount } from 'svelte';

  let apiStatus = 'Checking local API…';
  let agentCount = 0;

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
  <aside class="sidebar">
    <div class="brand"><span class="mark">◎</span><span>Confignostic</span></div>
    <div class="navtitle">WORKSPACE</div>
    <div class="nav active">⌘ &nbsp; Unified chat</div>
    <div class="nav">◇ &nbsp; Workflows <span class="soon">planned</span></div>
    <div class="nav">◈ &nbsp; Agents <span class="soon">planned</span></div>
    <div class="sidebar-footer">Open-source developer scaffold<br /><small>{apiStatus}</small></div>
  </aside>

  <main class="main">
    <header><strong>Unified chat</strong><span class="tag">Scaffold · not operational</span></header>
    <section class="welcome">
      <div class="orb">◎</div>
      <h1>One workspace for all your AI agents.</h1>
      <p>Connect existing agents, coordinate work, and see execution across providers. This is an initial interface shell, not a functional orchestration system.</p>
      <div class="cards">
        <div class="card"><span class="card-title">Connections</span><b>{agentCount} agents</b><small>No adapters are installed</small></div>
        <div class="card"><span class="card-title">Orchestration</span><b>Visual canvas</b><small>Planned feature</small></div>
        <div class="card"><span class="card-title">Execution</span><b>Durable tasks</b><small>Planned feature</small></div>
      </div>
    </section>
    <div class="composer">
      <input disabled aria-label="Chat unavailable" placeholder="Connect an agent to enable chat (coming soon)" />
      <button disabled aria-label="Sending unavailable">↗</button>
    </div>
    <p class="note">No agent messages have been sent. The backend currently serves only health and empty listings.</p>
  </main>
</div>
