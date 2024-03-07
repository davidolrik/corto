<script>
  import { api } from './lib/api.js'
  import { auth, logout, switchTenant } from './lib/auth.svelte.js'
  import { router } from './lib/router.svelte.js'
  import Login from './lib/pages/Login.svelte'
  import Dashboard from './lib/pages/Dashboard.svelte'
  import ShortCodes from './lib/pages/ShortCodes.svelte'
  import Domains from './lib/pages/Domains.svelte'
  import Tags from './lib/pages/Tags.svelte'
  import Profile from './lib/pages/Profile.svelte'

  let version = $state('')
  api('/api/version')
    .then((result) => (version = result.version))
    .catch(() => {})

  // "v" prefix only for numeric versions: "Corto v1.2.3" but "Corto devel"
  const footerLabel = $derived(version ? `Corto ${/^\d/.test(version) ? 'v' : ''}${version}` : 'Corto')

  let tenantMenu = $state(null)
  let switchError = $state('')

  async function selectTenant(slug) {
    if (tenantMenu) {
      tenantMenu.open = false
    }
    if (slug === auth.tenantSlug) {
      return
    }
    try {
      await switchTenant(slug)
      router.path = '/'
      location.hash = '#/'
    } catch (err) {
      switchError = err.message
    }
  }
</script>

{#if !auth.token}
  <Login />
{:else}
  <header class="topbar">
    <span class="brand" aria-label="Corto">🔗</span>
    <nav>
      <a href="#/" class:active={router.path === '/'}>{auth.tenantName || 'Dashboard'}</a>
      <a href="#/links" class:active={router.path === '/links'}>Links</a>
      <a href="#/domains" class:active={router.path === '/domains'}>Domains</a>
      <a href="#/tags" class:active={router.path === '/tags'}>Tags</a>
    </nav>
    <span class="user-area">
      {#if auth.tenants.length > 1}
        <details class="tenant-menu" bind:this={tenantMenu}>
          <summary aria-label="Switch tenant">{auth.tenantSlug}</summary>
          <div class="menu">
            {#each auth.tenants as tenant (tenant.slug)}
              <button class:active={tenant.slug === auth.tenantSlug} onclick={() => selectTenant(tenant.slug)}>
                {tenant.slug}
              </button>
            {/each}
          </div>
        </details>
      {:else}
        <span class="tenant-label">{auth.tenantSlug}</span>
      {/if}
      <span class="sep">/</span>
      <a class="profile-link" href="#/profile">{auth.username || 'Profile'}</a>
    </span>
    <button onclick={logout}>Log out</button>
  </header>
  {#if switchError}
    <div class="error">{switchError}</div>
  {/if}
  {#key auth.tenantSlug}
    <main>
      {#if router.path === '/'}
        <Dashboard />
      {:else if router.path === '/links'}
        <ShortCodes />
      {:else if router.path === '/domains'}
        <Domains />
      {:else if router.path === '/tags'}
        <Tags />
      {:else if router.path === '/profile'}
        <Profile />
      {:else}
        <p class="empty">Page not found</p>
      {/if}
    </main>
  {/key}
{/if}
<footer>{footerLabel}</footer>
