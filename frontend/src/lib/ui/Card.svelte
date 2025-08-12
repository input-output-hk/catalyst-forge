<script lang="ts">
	type Variant = 'default' | 'stat';
	let { title = '', children, variant = 'default' as Variant } = $props();
	const isStat = variant === 'stat';
	const base = 'cf-card';
	const classes = `${base} ${isStat ? base + '--stat' : ''}`;
</script>

<section class={classes}>
	{#if title}
		<header class="cf-card__header"><h3>{title}</h3></header>
	{/if}
	<div class="cf-card__body">
		{@render children?.()}
	</div>
</section>

<style>
	.cf-card {
		background: var(--cf-surface);
		border-radius: var(--cf-radius-md);
		box-shadow: var(--cf-shadow-sm);
		padding: var(--space-5);
		border: 1px solid var(--cf-border-subtle);
	}
	.cf-card__header h3 {
		margin: 0 0 0.5rem 0;
		color: var(--cf-blue-500);
	}
	.cf-card__body {
		color: var(--cf-ink);
	}
	.cf-card__footer {
		margin-top: 0.75rem;
	}

	/* Stat variant */
	.cf-card--stat {
		display: grid;
		gap: var(--space-2);
		align-items: start;
	}
	.cf-card--stat .cf-card__header h3 {
		color: var(--cf-text-subtle);
		font-weight: var(--weight-medium);
		font-size: 0.95rem;
	}
	.cf-card--stat .cf-card__body {
		font-weight: var(--weight-bold);
		font-size: clamp(1.5rem, 1rem + 2vw, 2rem);
		letter-spacing: -0.02em;
		color: var(--cf-text);
	}
</style>
