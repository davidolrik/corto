<script>
  import world from '@svg-maps/world'
  import { countryFlag } from '../country.js'

  let { visitsByCountry = {} } = $props()

  const max = $derived(Math.max(1, ...world.locations.map((l) => visitsByCountry[l.id.toUpperCase()] ?? 0)))
  const total = $derived(Object.values(visitsByCountry).reduce((sum, count) => sum + count, 0))

  let container = $state(null)
  let hovered = $state(null) // { location, x, y }

  function visits(location) {
    return visitsByCountry[location.id.toUpperCase()] ?? 0
  }

  // Log scale keeps small countries visible next to dominant ones
  function intensity(count) {
    if (count === 0) {
      return 0
    }
    return 0.3 + (0.7 * Math.log(count + 1)) / Math.log(max + 1)
  }

  function label(location) {
    const count = visits(location)
    return `${location.name}: ${count} ${count === 1 ? 'visit' : 'visits'}`
  }

  function tooltipStats(location) {
    const count = visits(location)
    let stats = `${count} ${count === 1 ? 'visit' : 'visits'}`
    if (count > 0 && total > 0) {
      stats += ` · ${Math.round((count / total) * 100)}%`
    }
    return stats
  }

  function track(event, location) {
    const rect = container.getBoundingClientRect()
    hovered = { location, x: event.clientX - rect.left, y: event.clientY - rect.top }
  }
</script>

<div class="map-container" bind:this={container}>
  <svg viewBox={world.viewBox} role="img" aria-label="Visits by country">
    {#each world.locations as location (location.id)}
      <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
      <path
        d={location.path}
        class="country"
        class:visited={visits(location) > 0}
        fill-opacity={visits(location) > 0 ? intensity(visits(location)) : 1}
        aria-label={label(location)}
        onmouseenter={(event) => track(event, location)}
        onmousemove={(event) => track(event, location)}
        onmouseleave={() => (hovered = null)}
      />
    {/each}
  </svg>
  {#if hovered}
    <div class="map-tooltip" style={`left: ${hovered.x}px; top: ${hovered.y}px`}>
      {#if countryFlag(hovered.location.id)}
        <span class="tooltip-flag">{countryFlag(hovered.location.id)}</span>
      {/if}
      <span class="tooltip-name">{hovered.location.name}</span>
      <span class="tooltip-stats">{tooltipStats(hovered.location)}</span>
    </div>
  {/if}
</div>

<style>
  .map-container {
    position: relative;
  }

  svg {
    width: 100%;
    height: auto;
    display: block;
  }

  .country {
    fill: var(--border);
    stroke: var(--surface);
    stroke-width: 0.5;
  }

  .country.visited {
    fill: var(--accent);
  }

  .map-tooltip {
    position: absolute;
    transform: translate(-50%, calc(-100% - 10px));
    display: flex;
    align-items: center;
    gap: 0.4rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    padding: 0.35rem 0.7rem;
    font-size: 0.85rem;
    white-space: nowrap;
    pointer-events: none;
    z-index: 30;
  }

  .tooltip-name {
    font-weight: 600;
  }

  .tooltip-stats {
    color: var(--muted);
  }
</style>
