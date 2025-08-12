<script lang="ts">
	import Button from '$lib/ui/Button.svelte';
	let { form } = $props();
</script>

<section class="section container" style="max-width:720px;">
	<h1>Approve Device</h1>
	<p class="text-subtle">Enter the user code from the requesting device.</p>
	<form method="POST" class="stack-4">
		<div class="stack-2 grid gap-2">
			<label for="user_code">User Code</label>
			<input id="user_code" name="user_code" placeholder="ABCD-EFGH" required />
		</div>
		<Button type="submit">Approve</Button>
	</form>

	{#if form?.ok}
		<p>Approved. The requesting device will complete sign-in shortly.</p>
	{:else if form?.error}
		<p style="color:crimson;">{form.error}</p>
	{/if}
</section>
