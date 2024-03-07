<script>
  import { api } from '../api.js'
  import { auth } from '../auth.svelte.js'

  // The world map carries ~1MB of country path data; load it on demand so
  // the main bundle stays small
  const worldMap = import('../components/WorldMap.svelte')

  let stats = $state(null)
  let error = $state('')

  async function load() {
    try {
      stats = await api('/api/stats')
    } catch (err) {
      error = err.message
    }
  }

  load()
</script>

<div class="page-head">
  <h1>{auth.tenantName || 'Dashboard'}</h1>
</div>

{#if error}
  <div class="error">{error}</div>
{/if}

{#if stats}
  <div class="stat-cards">
    <div class="card stat-card">
      <span class="stat-number">{stats.visits}</span>
      <span class="stat-label">clicks total</span>
    </div>
    <div class="card stat-card">
      <span class="stat-number">{stats.visits_this_week}</span>
      <span class="stat-label">this week</span>
    </div>
    <div class="card stat-card">
      <span class="stat-number">{stats.links}</span>
      <span class="stat-label">links</span>
    </div>
    <div class="card stat-card">
      <span class="stat-number">{stats.domains}</span>
      <span class="stat-label">domains</span>
    </div>
    <div class="card stat-card">
      <span class="stat-number">{stats.tags}</span>
      <span class="stat-label">tags</span>
    </div>
  </div>

  <div class="card">
    {#await worldMap then { default: WorldMap }}
      <WorldMap visitsByCountry={stats.visits_by_country} />
    {/await}
    {#if stats.visits_by_country.unknown}
      <p class="detail map-note">{stats.visits_by_country.unknown} clicks without a known location</p>
    {/if}
  </div>
{:else if !error}
  <p class="empty">Loading…</p>
{/if}
