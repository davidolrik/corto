<script>
  import { login } from '../api.js'
  import { setAuthenticated } from '../auth.svelte.js'

  let username = $state('')
  let password = $state('')
  let error = $state('')
  let busy = $state(false)

  async function submit(event) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      const result = await login(username, password)
      setAuthenticated(result)
    } catch (err) {
      error = err.message
    } finally {
      busy = false
    }
  }
</script>

<div class="login">
  <div class="card">
    <h1>🔗 Corto</h1>
    {#if error}
      <div class="error">{error}</div>
    {/if}
    <form onsubmit={submit}>
      <div class="field">
        <label for="username">Username</label>
        <input id="username" type="text" bind:value={username} autocomplete="username" required />
      </div>
      <div class="field">
        <label for="password">Password</label>
        <input id="password" type="password" bind:value={password} autocomplete="current-password" required />
      </div>
      <div class="form-actions">
        <button class="primary" type="submit" disabled={busy} style="width: 100%">Log in</button>
      </div>
    </form>
  </div>
</div>
