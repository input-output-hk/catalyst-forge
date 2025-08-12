<script lang="ts">
	let { open = false, children } = $props();
	let backdropEl: HTMLDivElement | null = null;
	function onBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) open = false;
	}
	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' || e.key === 'Enter') open = false;
	}
	$effect(() => {
		if (open && backdropEl) backdropEl.focus();
	});
</script>

{#if open}
	<div
		bind:this={backdropEl}
		class="cf-modal__backdrop"
		role="dialog"
		aria-modal="true"
		onclick={onBackdropClick}
		onkeydown={onKeydown}
		tabindex="-1"
	>
		<div class="cf-modal__content" role="document">
			{@render children?.()}
		</div>
	</div>
{/if}

<style>
	.cf-modal__backdrop {
		position: fixed;
		inset: 0;
		background: rgba(11, 18, 32, 0.4);
		display: grid;
		place-items: center;
	}
	.cf-modal__content {
		background: var(--cf-bg);
		border-radius: var(--cf-radius-md);
		box-shadow: var(--cf-shadow-md);
		padding: 1rem;
		min-width: min(92vw, 520px);
	}
</style>
