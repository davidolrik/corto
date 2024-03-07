<script>
  import { api } from '../api.js'
  import Modal from '../components/Modal.svelte'

  let domains = $state([])
  let error = $state('')
  let loading = $state(true)

  let editing = $state(null) // null = closed, '' = new, public_id = editing
  let form = $state({ fqdn: '', fallback_url: '', description: '' })

  async function load() {
    loading = true
    try {
      domains = await api('/api/domains')
      error = ''
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  function openNew() {
    form = { fqdn: '', fallback_url: '', description: '' }
    editing = ''
  }

  function openEdit(domain) {
    form = {
      fqdn: domain.fqdn,
      fallback_url: domain.fallback_url || '',
      description: domain.description || '',
    }
    editing = domain.public_id
  }

  async function save(event) {
    event.preventDefault()
    try {
      if (editing === '') {
        await api('/api/domains', { method: 'POST', body: form })
      } else {
        await api(`/api/domains/${editing}`, { method: 'PUT', body: form })
      }
      editing = null
      await load()
    } catch (err) {
      error = err.message
    }
  }

  async function remove(domain) {
    if (!confirm(`Delete domain "${domain.fqdn}"?`)) {
      return
    }
    try {
      await api(`/api/domains/${domain.public_id}`, { method: 'DELETE' })
      await load()
    } catch (err) {
      error = err.message
    }
  }

  load()
</script>

<div class="page-head">
  <h1>Domains</h1>
  <button class="primary" onclick={openNew}>New domain</button>
</div>

{#if error}
  <div class="error">{error}</div>
{/if}

{#if editing !== null}
  <Modal title={editing === '' ? 'New domain' : 'Edit domain'} onclose={() => (editing = null)}>
    <form onsubmit={save}>
      <div class="field">
        <label for="d-fqdn">Domain (FQDN)</label>
        <input id="d-fqdn" type="text" bind:value={form.fqdn} required placeholder="go.example.com" />
      </div>
      <div class="field">
        <label for="d-fallback">Fallback URL</label>
        <input id="d-fallback" type="url" bind:value={form.fallback_url} placeholder="Where unknown slugs are sent" />
      </div>
      <div class="field">
        <label for="d-description">Description</label>
        <input id="d-description" type="text" bind:value={form.description} placeholder="What this domain is for" />
      </div>
      <div class="form-actions">
        <button class="primary" type="submit">{editing === '' ? 'Create' : 'Save'}</button>
        <button type="button" onclick={() => (editing = null)}>Cancel</button>
      </div>
    </form>
  </Modal>
{/if}

{#if loading}
  <p class="empty">Loading…</p>
{:else if domains.length === 0}
  <p class="empty">No domains yet. Add the domain your short links will live on.</p>
{:else}
  <div class="list">
    {#each domains as domain (domain.public_id)}
      <div class="row">
        <div class="info">
          <div class="title">{domain.fqdn}</div>
          {#if domain.description}
            <div class="detail">{domain.description}</div>
          {/if}
          {#if domain.fallback_url}
            <div class="detail">↳ fallback: {domain.fallback_url}</div>
          {/if}
        </div>
        <div class="actions">
          <button onclick={() => openEdit(domain)}>Edit</button>
          <button class="danger" onclick={() => remove(domain)}>Delete</button>
        </div>
      </div>
    {/each}
  </div>
{/if}
