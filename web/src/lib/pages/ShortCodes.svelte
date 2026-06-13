<script>
  import { api } from '../api.js'
  import { readableOnDark, darkTint } from '../color.js'
  import { countryFlag } from '../country.js'
  import { toDateTimeInput, fromDateTimeInput } from '../datetime.js'
  import Modal from '../components/Modal.svelte'

  // The world map is heavy; load it on demand when a row is expanded
  const worldMap = import('../components/WorldMap.svelte')

  let shortCodes = $state([])
  let availableDomains = $state([])
  let availableTags = $state([])
  let error = $state('')
  let loading = $state(true)

  let editing = $state(null) // null = closed, '' = new, public_id = editing
  let expanded = $state(null) // public_id of the row with stats expanded
  let copied = $state(null) // public_id whose link was just copied
  let selectedDomains = $state([]) // domains chosen as filters (FQDNs)
  let selectedTags = $state([]) // tags chosen as filters (slugs)
  let filterQuery = $state('') // current text in the typeahead box
  let filterInput = $state(null) // the typeahead <input>, for refocusing
  let activeIndex = $state(-1) // highlighted suggestion, -1 = none
  let menuEl = $state(null) // the suggestion <ul>, for scrolling the active row
  let form = $state(emptyForm())

  function emptyForm() {
    return {
      slug: '',
      title: '',
      description: '',
      target_url: '',
      fallback_url: '',
      forward_query: false,
      is_crawlable: false,
      valid_since: '',
      valid_until: '',
      max_visits: '',
      domains: [],
      tags: [],
    }
  }

  async function load() {
    loading = true
    // Load independently so one failing request doesn't discard the others
    const problems = []
    const fetch = (path) =>
      api(path).catch((err) => {
        problems.push(err.message)
        return null
      })
    const [codes, domains, tags] = await Promise.all([
      fetch('/api/short-codes'),
      fetch('/api/domains'),
      fetch('/api/tags'),
    ])
    if (codes) shortCodes = codes
    if (domains) availableDomains = domains
    if (tags) availableTags = tags
    error = problems.join(' · ')
    loading = false
  }

  function openNew() {
    form = emptyForm()
    editing = ''
  }

  function openEdit(sc) {
    form = {
      slug: sc.slug,
      title: sc.title || '',
      description: sc.description || '',
      target_url: sc.target_url,
      fallback_url: sc.fallback_url || '',
      forward_query: sc.forward_query,
      is_crawlable: sc.is_crawlable,
      valid_since: toDateTimeInput(sc.valid_since),
      valid_until: toDateTimeInput(sc.valid_until),
      max_visits: sc.max_visits ?? '',
      domains: [...sc.domains],
      tags: [...sc.tags],
    }
    editing = sc.public_id
  }

  async function save(event) {
    event.preventDefault()
    if (form.domains.length === 0) {
      error = 'Select at least one domain — a link without a domain is unreachable.'
      return
    }
    const body = {
      slug: form.slug,
      title: form.title,
      description: form.description,
      target_url: form.target_url,
      fallback_url: form.fallback_url,
      forward_query: form.forward_query,
      is_crawlable: form.is_crawlable,
      valid_since: fromDateTimeInput(form.valid_since),
      valid_until: fromDateTimeInput(form.valid_until),
      max_visits: form.max_visits === '' ? undefined : Number(form.max_visits),
      domains: form.domains,
      tags: form.tags,
    }
    try {
      if (editing === '') {
        await api('/api/short-codes', { method: 'POST', body })
      } else {
        await api(`/api/short-codes/${editing}`, { method: 'PUT', body })
      }
      editing = null
      await load()
    } catch (err) {
      error = err.message
    }
  }

  // Tag slug → color, for coloring the tag chips
  const tagColors = $derived(
    Object.fromEntries(availableTags.filter((t) => t.color).map((t) => [t.slug, t.color]))
  )

  // The list is filtered down cumulatively: a code must carry every selected
  // domain and every selected tag (AND across all tokens). Each token added
  // narrows the result further.
  const filtered = $derived(
    shortCodes.filter(
      (sc) =>
        selectedDomains.every((d) => sc.domains.includes(d)) &&
        selectedTags.every((t) => sc.tags.includes(t))
    )
  )

  // Suggestions are drawn from the already-filtered codes, so every option
  // narrows the list to something non-empty. Already-chosen values are dropped
  // and the query matches as a substring (a leading "#" is tolerated).
  const suggestions = $derived.by(() => {
    const query = filterQuery.trim().toLowerCase().replace(/^#+/, '')
    if (!query) {
      return []
    }
    const domains = [...new Set(filtered.flatMap((sc) => sc.domains))]
      .filter((d) => !selectedDomains.includes(d) && d.toLowerCase().includes(query))
      .sort()
      .map((value) => ({ type: 'domain', value }))
    const tags = [...new Set(filtered.flatMap((sc) => sc.tags))]
      .filter((t) => !selectedTags.includes(t) && t.toLowerCase().includes(query))
      .sort()
      .map((value) => ({ type: 'tag', value }))
    return [...domains, ...tags]
  })

  function addFilter(suggestion) {
    if (suggestion.type === 'domain') {
      if (!selectedDomains.includes(suggestion.value)) {
        selectedDomains = [...selectedDomains, suggestion.value]
      }
    } else if (!selectedTags.includes(suggestion.value)) {
      selectedTags = [...selectedTags, suggestion.value]
    }
    filterQuery = ''
    activeIndex = -1
    filterInput?.focus()
  }

  function removeDomain(fqdn) {
    selectedDomains = selectedDomains.filter((d) => d !== fqdn)
  }

  function removeTag(slug) {
    selectedTags = selectedTags.filter((t) => t !== slug)
  }

  // Keep the highlighted suggestion scrolled into view as the arrows move it
  function scrollActiveIntoView() {
    const options = menuEl?.querySelectorAll('.typeahead-option')
    options?.[activeIndex]?.scrollIntoView?.({ block: 'nearest' })
  }

  function filterKeydown(event) {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      if (suggestions.length > 0) {
        activeIndex = (activeIndex + 1) % suggestions.length
        scrollActiveIntoView()
      }
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      if (suggestions.length > 0) {
        activeIndex = (activeIndex - 1 + suggestions.length) % suggestions.length
        scrollActiveIntoView()
      }
    } else if (event.key === 'Enter') {
      event.preventDefault()
      if (suggestions.length > 0) {
        addFilter(suggestions[activeIndex >= 0 ? activeIndex : 0])
      }
    } else if (event.key === 'Tab' && !event.shiftKey && suggestions.length > 0) {
      // Tab commits the highlighted (or first) suggestion instead of leaving
      event.preventDefault()
      addFilter(suggestions[activeIndex >= 0 ? activeIndex : 0])
    } else if (event.key === 'Escape') {
      filterQuery = ''
      activeIndex = -1
    } else if (event.key === 'Backspace' && filterQuery === '') {
      // Pop the last token, matching the on-screen order (tags after domains)
      if (selectedTags.length > 0) {
        selectedTags = selectedTags.slice(0, -1)
      } else if (selectedDomains.length > 0) {
        selectedDomains = selectedDomains.slice(0, -1)
      }
    }
  }

  // Breakdown entries, biggest bucket first
  function statEntries(breakdown) {
    return Object.entries(breakdown ?? {}).sort((a, b) => b[1] - a[1])
  }

  // Two-letter ISO country codes become flag emoji; other buckets (like
  // "unknown") get no flag
  function countryLabel(country) {
    const flag = countryFlag(country)
    return flag ? `${flag} ${country}` : country
  }

  function toggleStats(sc) {
    expanded = expanded === sc.public_id ? null : sc.public_id
  }

  function rowKeydown(event, sc) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      toggleStats(sc)
    }
  }

  function editClicked(event, sc) {
    event.stopPropagation()
    openEdit(sc)
  }

  function copyLink(event, sc, fqdn) {
    event.stopPropagation()
    navigator.clipboard?.writeText(`https://${fqdn}/${sc.slug}`)
    const key = `${sc.public_id}:${fqdn}`
    copied = key
    setTimeout(() => {
      if (copied === key) {
        copied = null
      }
    }, 1500)
  }

  async function remove(event, sc) {
    event.stopPropagation()
    if (!confirm(`Delete link "${sc.slug}"?`)) {
      return
    }
    try {
      await api(`/api/short-codes/${sc.public_id}`, { method: 'DELETE' })
      await load()
    } catch (err) {
      error = err.message
    }
  }

  load()
</script>

<div class="page-head">
  <h1>Links</h1>
  <button class="primary" onclick={openNew}>New link</button>
</div>

{#if error}
  <div class="error">{error}</div>
{/if}

{#if editing !== null}
  <Modal title={editing === '' ? 'New link' : 'Edit link'} onclose={() => (editing = null)}>
    <form onsubmit={save}>
      <div class="field-grid">
        <div class="field">
          <label for="sc-slug">Slug</label>
          <input
            id="sc-slug"
            type="text"
            bind:value={form.slug}
            required={editing !== ''}
            placeholder={editing === '' ? 'leave empty to generate' : 'promo'}
          />
        </div>
        <div class="field">
          <label for="sc-title">Title</label>
          <input id="sc-title" type="text" bind:value={form.title} placeholder="Spring promo" />
        </div>
      </div>
      <div class="field">
        <label for="sc-description">Description</label>
        <input id="sc-description" type="text" bind:value={form.description} placeholder="What this link is for" />
      </div>
      <div class="field">
        <label for="sc-target">Target URL</label>
        <input id="sc-target" type="url" bind:value={form.target_url} required placeholder="https://example.com/landing" />
      </div>
      <div class="field">
        <label for="sc-fallback">Fallback URL</label>
        <input id="sc-fallback" type="url" bind:value={form.fallback_url} placeholder="Used outside the validity window" />
      </div>
      <div class="field-grid">
        <div class="field">
          <label for="sc-since">Valid since</label>
          <input id="sc-since" type="datetime-local" bind:value={form.valid_since} />
        </div>
        <div class="field">
          <label for="sc-until">Valid until</label>
          <input id="sc-until" type="datetime-local" bind:value={form.valid_until} />
        </div>
        <div class="field">
          <label for="sc-max-visits">Max visits</label>
          <input id="sc-max-visits" type="number" min="1" bind:value={form.max_visits} placeholder="unlimited" />
        </div>
      </div>
      <div class="field">
        <label for="sc-domains">Domains</label>
        <div class="checks" id="sc-domains">
          {#each availableDomains as domain (domain.public_id)}
            <label>
              <input type="checkbox" bind:group={form.domains} value={domain.fqdn} />
              {domain.fqdn}
            </label>
          {:else}
            <span class="detail">No domains yet — create one under Domains first.</span>
          {/each}
        </div>
      </div>
      <div class="field">
        <label for="sc-tags">Tags</label>
        <div class="checks" id="sc-tags">
          {#each availableTags as tag (tag.public_id)}
            <label>
              <input type="checkbox" bind:group={form.tags} value={tag.slug} />
              {tag.slug}
            </label>
          {:else}
            <span class="detail">No tags yet.</span>
          {/each}
        </div>
      </div>
      <div class="field checks">
        <label>
          <input type="checkbox" bind:checked={form.forward_query} />
          Forward query string
        </label>
        <label>
          <input type="checkbox" bind:checked={form.is_crawlable} />
          Crawlable (robots.txt)
        </label>
      </div>
      <div class="form-actions">
        <button class="primary" type="submit">{editing === '' ? 'Create' : 'Save'}</button>
        <button type="button" onclick={() => (editing = null)}>Cancel</button>
      </div>
    </form>
  </Modal>
{/if}

{#if !loading && shortCodes.length > 0}
  <div class="list-filter">
    <div class="filter-tokens">
      {#each selectedDomains as fqdn (fqdn)}
        <span class="chip domain token">
          <span>{fqdn}</span>
          <button
            type="button"
            class="token-remove"
            aria-label={`Remove domain ${fqdn}`}
            onclick={() => removeDomain(fqdn)}
          >×</button>
        </span>
      {/each}
      {#each selectedTags as slug (slug)}
        <span
          class="chip token"
          style={tagColors[slug]
            ? `background-color: ${darkTint(tagColors[slug])}; border-color: ${readableOnDark(tagColors[slug])}; color: ${readableOnDark(tagColors[slug])}`
            : ''}
        >
          <span>#{slug}</span>
          <button
            type="button"
            class="token-remove"
            aria-label={`Remove tag ${slug}`}
            onclick={() => removeTag(slug)}
          >×</button>
        </span>
      {/each}
      <input
        type="text"
        class="filter-input"
        bind:this={filterInput}
        bind:value={filterQuery}
        oninput={() => (activeIndex = -1)}
        onkeydown={filterKeydown}
        aria-label="Filter links"
        placeholder="Filter by domain or tag…"
      />
    </div>
    {#if suggestions.length > 0}
      <ul class="typeahead-menu" aria-label="Filter suggestions" bind:this={menuEl}>
        {#each suggestions as suggestion, index (suggestion.type + ':' + suggestion.value)}
          <li>
            <button
              type="button"
              class="typeahead-option"
              class:active={index === activeIndex}
              onclick={() => addFilter(suggestion)}
              onmouseenter={() => (activeIndex = index)}
            >
              {#if suggestion.type === 'domain'}
                <span class="chip domain">{suggestion.value}</span>
              {:else}
                <span
                  class="chip"
                  style={tagColors[suggestion.value]
                    ? `background-color: ${darkTint(tagColors[suggestion.value])}; border-color: ${readableOnDark(tagColors[suggestion.value])}; color: ${readableOnDark(tagColors[suggestion.value])}`
                    : ''}
                >#{suggestion.value}</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
{/if}

{#if loading}
  <p class="empty">Loading…</p>
{:else if shortCodes.length === 0}
  <p class="empty">No links yet. Create your first one!</p>
{:else if filtered.length === 0}
  <p class="empty">No links match the filter.</p>
{:else}
  <div class="list">
    {#each filtered as sc (sc.public_id)}
      <div
        class="row clickable expandable"
        class:expanded={expanded === sc.public_id}
        role="button"
        tabindex="0"
        aria-expanded={expanded === sc.public_id}
        onclick={() => toggleStats(sc)}
        onkeydown={(event) => rowKeydown(event, sc)}
      >
        <div class="row-main">
        <div class="info">
          <div class="title">
            <span class="slug">/{sc.slug}</span>
            {#if sc.title}· {sc.title}{/if}
          </div>
          {#if sc.description}
            <div class="detail">{sc.description}</div>
          {/if}
          <div class="detail">→ {sc.target_url}</div>
          {#if sc.domains.length || sc.tags.length}
            <div class="chips">
              {#each sc.domains as fqdn (fqdn)}
                <button
                  class="chip domain"
                  title={`Copy https://${fqdn}/${sc.slug}`}
                  onclick={(event) => copyLink(event, sc, fqdn)}
                >
                  {copied === `${sc.public_id}:${fqdn}` ? 'Copied' : fqdn}
                </button>
              {/each}
              {#each sc.tags as tag}
                <span
                  class="chip"
                  style={tagColors[tag]
                    ? `background-color: ${darkTint(tagColors[tag])}; border-color: ${readableOnDark(tagColors[tag])}; color: ${readableOnDark(tagColors[tag])}`
                    : ''}
                >#{tag}</span>
              {/each}
            </div>
          {/if}
        </div>
        <div class="stats">
          <span class="stat">
            <span class="stat-number">{sc.visits_this_week}</span>
            <span class="stat-label">this week</span>
          </span>
          <span class="stat">
            <span
              class="stat-number"
              class:exhausted={sc.max_visits != null && sc.visits >= sc.max_visits}
            >{sc.visits}{sc.max_visits != null ? ` / ${sc.max_visits}` : ''}</span>
            <span class="stat-label">total</span>
          </span>
        </div>
        <div class="actions">
          <button onclick={(event) => editClicked(event, sc)}>Edit</button>
          <button class="danger" onclick={(event) => remove(event, sc)}>Delete</button>
        </div>
        </div>
      {#if expanded === sc.public_id}
        <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
        <div class="row-stats" onclick={(event) => event.stopPropagation()}>
          <div class="row-stats-map">
            {#await worldMap then { default: WorldMap }}
              <WorldMap visitsByCountry={sc.visits_by_country} />
            {/await}
          </div>
          <div class="row-stats-sidebar">
            <h3>Domains</h3>
            <div class="breakdown">
              {#each statEntries(sc.visits_by_domain) as [fqdn, count] (fqdn)}
                <div class="breakdown-row"><span>{fqdn}</span><span class="count">{count}</span></div>
              {:else}
                <p class="detail">No visits yet.</p>
              {/each}
            </div>
            <h3>Campaigns</h3>
            <div class="breakdown">
              {#each statEntries(sc.visits_by_campaign) as [campaign, count] (campaign)}
                <div class="breakdown-row"><span>{campaign}</span><span class="count">{count}</span></div>
              {:else}
                <p class="detail">No visits yet.</p>
              {/each}
            </div>
            <h3>Countries</h3>
            <div class="breakdown">
              {#each statEntries(sc.visits_by_country) as [country, count] (country)}
                <div class="breakdown-row"><span>{countryLabel(country)}</span><span class="count">{count}</span></div>
              {:else}
                <p class="detail">No visits yet.</p>
              {/each}
            </div>
          </div>
        </div>
      {/if}
      </div>
    {/each}
  </div>
{/if}
