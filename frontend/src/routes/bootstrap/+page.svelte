<script lang="ts">
  import Button from '$lib/ui/Button.svelte';
  import { browser } from '$app/environment';
  let { form } = $props();
  let token = $state('');
  let email = $state('');
  let errorMsg = $state<string | null>(null);

  $effect(() => {
    errorMsg = (form as any)?.error ?? null;
  });
</script>

<section class="section container" style="max-width:640px;">
  <h1>Bootstrap</h1>
  <p class="text-subtle">Enter the one-time bootstrap token to create an admin invite and start device registration.</p>

  {#if errorMsg}
    <p style="color:crimson;">{errorMsg}</p>
  {/if}

  <form method="POST" action="?/init" class="stack-4" on:submit={() => (errorMsg=null)}>
    <div class="stack-2">
      <label for="token">Bootstrap token</label>
      <input id="token" name="token" bind:value={token} placeholder="paste token" required />
    </div>
    <div class="stack-2">
      <label for="email">Admin email</label>
      <input id="email" type="email" name="email" bind:value={email} placeholder="admin@example.com" required />
    </div>
    <Button type="submit">Continue</Button>
  </form>

  {#if form?.auto && form?.init}
    <div class="card" style="padding:16px;margin-top:16px;">
      <div class="text-subtle">Finishing device registration…</div>
      {#if browser}
        {@html (() => {
          (async () => {
            try {
              const mod = await import('$lib/auth/device');
              await mod.registerDevice(form.init, 'This browser');
              location.href = '/';
            } catch (e) {
              console.error('Auto device register error', e);
              errorMsg = (e as Error).message;
            }
          })();
          return '';
        })()}
      {/if}
      <form on:submit|preventDefault={async () => {
        // lazy-import to avoid SSR issues
        const mod = await import('$lib/auth/device');
        try {
          await mod.registerDevice(form.init, 'This browser');
          // Redirect to home
          location.href = '/';
        } catch (e) {
          console.error('Device register error', e);
          alert('Registration failed: ' + (e as Error).message);
        }
      }}>
        <Button type="submit">Register device now</Button>
      </form>

      <div class="text-subtle" style="margin-top:12px;">After registration you will be redirected to the home page.</div>
    </div>
  {/if}
</section>


