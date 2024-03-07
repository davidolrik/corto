<script>
  import { api } from '../api.js'

  let profile = $state(null)
  let currentPassword = $state('')
  let newPassword = $state('')
  let message = $state('')
  let error = $state('')

  api('/api/profile')
    .then((result) => (profile = result))
    .catch((err) => (error = err.message))

  async function changePassword(event) {
    event.preventDefault()
    error = ''
    message = ''
    try {
      await api('/api/profile/password', {
        method: 'PUT',
        body: { current_password: currentPassword, new_password: newPassword },
      })
      message = 'Password changed'
      currentPassword = ''
      newPassword = ''
    } catch (err) {
      error = err.message
    }
  }
</script>

<div class="page-head">
  <h1>Profile</h1>
</div>

{#if error}
  <div class="error">{error}</div>
{/if}
{#if message}
  <div class="success">{message}</div>
{/if}

{#if profile}
  <p class="detail">Logged in as <strong>{profile.username}</strong></p>
{/if}

<div class="card">
  <h2>Change password</h2>
  <form onsubmit={changePassword}>
    <div class="field">
      <label for="p-current">Current password</label>
      <input
        id="p-current"
        type="password"
        bind:value={currentPassword}
        autocomplete="current-password"
        required
      />
    </div>
    <div class="field">
      <label for="p-new">New password</label>
      <input
        id="p-new"
        type="password"
        bind:value={newPassword}
        autocomplete="new-password"
        minlength="8"
        required
      />
    </div>
    <div class="form-actions">
      <button class="primary" type="submit">Change password</button>
    </div>
  </form>
</div>
