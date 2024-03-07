<script>
  import { api } from '../api.js'
  import Modal from '../components/Modal.svelte'

  let tags = $state([])
  let error = $state('')
  let loading = $state(true)

  const DEFAULT_COLOR = '#4f46e5'

  let editing = $state(null) // null = closed, '' = new, public_id = editing
  let form = $state({ slug: '', color: DEFAULT_COLOR, description: '' })

  async function load() {
    loading = true
    try {
      tags = await api('/api/tags')
      error = ''
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  function openNew() {
    form = { slug: '', color: DEFAULT_COLOR, description: '' }
    editing = ''
  }

  function openEdit(tag) {
    form = { slug: tag.slug, color: tag.color || DEFAULT_COLOR, description: tag.description || '' }
    editing = tag.public_id
  }

  async function save(event) {
    event.preventDefault()
    try {
      if (editing === '') {
        await api('/api/tags', { method: 'POST', body: form })
      } else {
        await api(`/api/tags/${editing}`, { method: 'PUT', body: form })
      }
      editing = null
      await load()
    } catch (err) {
      error = err.message
    }
  }

  async function remove(tag) {
    if (!confirm(`Delete tag "${tag.slug}"?`)) {
      return
    }
    try {
      await api(`/api/tags/${tag.public_id}`, { method: 'DELETE' })
      await load()
    } catch (err) {
      error = err.message
    }
  }

  load()
</script>

<div class="page-head">
  <h1>Tags</h1>
  <button class="primary" onclick={openNew}>New tag</button>
</div>

{#if error}
  <div class="error">{error}</div>
{/if}

{#if editing !== null}
  <Modal title={editing === '' ? 'New tag' : 'Edit tag'} onclose={() => (editing = null)}>
    <form onsubmit={save}>
      <div class="field">
        <label for="t-slug">Tag</label>
        <input id="t-slug" type="text" bind:value={form.slug} required placeholder="campaign-2026" />
      </div>
      <div class="field">
        <label for="t-color">Color</label>
        <input id="t-color" type="color" bind:value={form.color} />
      </div>
      <div class="field">
        <label for="t-description">Description</label>
        <input id="t-description" type="text" bind:value={form.description} placeholder="What this tag is for" />
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
{:else if tags.length === 0}
  <p class="empty">No tags yet. Tags help you group and find links.</p>
{:else}
  <div class="list">
    {#each tags as tag (tag.public_id)}
      <div class="row tag-row" style={tag.color ? `border-left-color: ${tag.color}` : ''}>
        <div class="info">
          <div class="title">#{tag.slug}</div>
          {#if tag.description}
            <div class="detail">{tag.description}</div>
          {/if}
        </div>
        <div class="actions">
          <button onclick={() => openEdit(tag)}>Edit</button>
          <button class="danger" onclick={() => remove(tag)}>Delete</button>
        </div>
      </div>
    {/each}
  </div>
{/if}
