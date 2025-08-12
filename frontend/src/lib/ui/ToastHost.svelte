<script lang="ts">
	import { toasts } from './toast.store';
	import type { Toast } from './toast.store';
	let list: Toast[] = $state([]);
	$effect(() => {
		const unsub = toasts.subscribe((v) => (list = v));
		return () => unsub();
	});
</script>

<div class="cf-toast__host" aria-live="polite" aria-atomic="true">
	{#each list as t (t.id)}
		<div class="cf-toast">{t.text}</div>
	{/each}
	<slot />
</div>

<style>
	.cf-toast__host {
		position: fixed;
		bottom: 16px;
		right: 16px;
		display: grid;
		gap: 8px;
	}
	.cf-toast {
		background: var(--cf-ink);
		color: var(--cf-bg);
		padding: 0.5rem 0.75rem;
		border-radius: var(--cf-radius-sm);
		box-shadow: var(--cf-shadow-sm);
		max-width: min(92vw, 420px);
	}
</style>
