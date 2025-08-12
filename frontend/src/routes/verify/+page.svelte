<script lang="ts">
	let { data } = $props();
</script>

<section class="section container" style="max-width:720px;">
	<h1>Invite Verification</h1>
	{#if data?.ok}
		<p>Your invite was verified successfully.</p>
		<p class="text-subtle">You can now continue to sign in or approve a device.</p>
		<p class="stack-3">
			<a class="nav-link" href="/login">Go to Login</a>
			<a class="nav-link" href="/auth/device">Approve a Device</a>
		</p>
	{:else}
		<p style="color:crimson;">{data?.message ?? 'Verification failed.'}</p>
		<p class="text-subtle">Request a new invite from your administrator.</p>
	{/if}
</section>
