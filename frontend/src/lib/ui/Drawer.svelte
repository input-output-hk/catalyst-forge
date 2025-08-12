<script lang="ts">
	type Side = 'left' | 'right';
	let { open = false, side = 'right' as Side, children } = $props();
	function onBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) open = false;
	}
	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') open = false;
	}
</script>

{#if open}
	<div
		class="cf-drawer__backdrop"
		role="dialog"
		aria-modal="true"
		onclick={onBackdropClick}
		onkeydown={onKeydown}
		tabindex="-1"
	>
		<div class={`cf-drawer__panel cf-drawer__panel--${side}`}>
			{@render children?.()}
		</div>
	</div>
{/if}

<style>
	.cf-drawer__backdrop {
		position: fixed;
		inset: 0;
		background: rgba(11, 18, 32, 0.4);
		z-index: 1000;
	}
	.cf-drawer__panel {
		position: fixed;
		top: 0;
		bottom: 0;
		width: min(92vw, 420px);
		background: var(--cf-bg);
		box-shadow: var(--cf-shadow-md);
		padding: 1rem;
		overflow: auto;
		z-index: 1001;
	}
	.cf-drawer__panel--right {
		right: 0;
	}
	.cf-drawer__panel--left {
		left: 0;
	}
</style>
