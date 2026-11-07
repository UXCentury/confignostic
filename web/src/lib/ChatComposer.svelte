<div class="composer">
  <textarea disabled rows="2" aria-label="Chat unavailable" placeholder="Message an agent…"></textarea>
  <div class="toolbar"><span>Chat execution is planned</span><button disabled aria-label="Sending unavailable">↑</button></div>
</div>

<style>
  .composer { border: 1px solid #444; border-radius: 12px; background: var(--surface); padding: 14px 16px; margin-bottom: max(18px, env(safe-area-inset-bottom)); }
  textarea { width: 100%; resize: none; background: transparent; border: 0; font: inherit; }
  textarea::placeholder { color: var(--muted); opacity: 1; }
  .toolbar { display: flex; justify-content: space-between; align-items: center; color: var(--muted); font-size: 12px; }
  button { border: 0; background: #3d3d3d; color: #bbb; border-radius: 6px; width: 28px; height: 28px; font-size: 18px; }
</style>
