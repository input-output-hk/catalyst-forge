<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	type Variant = 'primary' | 'secondary' | 'ghost';
	type Size = 'sm' | 'md';

	let {
		children,
		variant = 'primary' as Variant,
		size = 'md' as Size,
		disabled = false,
		type = 'button' as 'button' | 'submit' | 'reset'
	} = $props();

	const base = 'cf-btn';

	const variantClass = `${base}--${variant}`;
	const sizeClass = `${base}--${size}`;

	const dispatch = createEventDispatcher<{ click: MouseEvent }>();
	function handleClick(event: MouseEvent) {
		dispatch('click', event);
	}
</script>

<button class={`${base} ${variantClass} ${sizeClass}`} {type} {disabled} onclick={handleClick}>
	{@render children?.()}
</button>

<style>
	.cf-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.5rem 0.875rem;
		border-radius: var(--cf-radius-sm);
		border: 1px solid transparent;
		cursor: pointer;
		font-weight: 600;
		line-height: 1.2;
		transition:
			box-shadow 150ms ease,
			background-color 150ms ease,
			border-color 150ms ease,
			transform 80ms ease;
	}

	.cf-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.cf-btn:focus-visible {
		outline: none;
		box-shadow: 0 0 0 3px color-mix(in oklab, var(--cf-blue-500) 40%, white);
	}

	.cf-btn--sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.9rem;
	}
	.cf-btn--md {
		padding: 0.5rem 0.875rem;
		font-size: 1rem;
	}

	.cf-btn--primary {
		background: var(--cf-blue-600);
		color: white;
		border-color: var(--cf-blue-600);
	}
	.cf-btn--primary:hover {
		filter: brightness(0.98);
	}
	.cf-btn--primary:active {
		transform: translateY(1px);
	}

	.cf-btn--secondary {
		background: transparent;
		color: var(--cf-blue-500);
		border-color: var(--cf-blue-500);
	}
	.cf-btn--secondary:hover {
		background: color-mix(in oklab, var(--cf-blue-500) 10%, white);
	}
	.cf-btn--secondary:active {
		transform: translateY(1px);
	}

	.cf-btn--ghost {
		background: transparent;
		color: var(--cf-ink);
		border-color: transparent;
	}
	.cf-btn--ghost:hover {
		background: var(--cf-bg-soft);
	}
	.cf-btn--ghost:active {
		transform: translateY(1px);
	}

	@media (prefers-reduced-motion: reduce) {
		.cf-btn {
			transition: none;
		}
		.cf-btn:active {
			transform: none;
		}
	}
</style>
