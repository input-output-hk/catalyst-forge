<script lang="ts">
  import Button from '$lib/ui/Button.svelte';
  let busy = $state(false);
  async function doLogout() {
    if (busy) return;
    busy = true;
    try {
      const { logout } = await import('$lib/auth/session');
      await logout();
      alert('Logged out.');
      location.href = '/login';
    } finally {
      busy = false;
    }
  }
</script>

<section class="section container" style="max-width:640px;">
  <h1>Logout</h1>
  <p>Revoke current refresh token and clear local state.</p>
  <Button on:click={doLogout} disabled={busy}>{busy ? 'Working…' : 'Logout'}</Button>
</section>



