<script lang="ts">
	let { children } = $props();
	const colors = [
		['cf-blue-600', 'var(--cf-blue-600)'],
		['cf-blue-500', 'var(--cf-blue-500)'],
		['cf-blue-300', 'var(--cf-blue-300)'],
		['cf-violet-300', 'var(--cf-violet-300)'],
		['cf-sky-200', 'var(--cf-sky-200)'],
		['cf-bg', 'var(--cf-bg)'],
		['cf-bg-soft', 'var(--cf-bg-soft)'],
		['cf-ink', 'var(--cf-ink)'],
		['cf-ink-muted', 'var(--cf-ink-muted)']
	];
</script>

<section class="container" style="padding: 2rem 0;">
	<h1>Brand Tokens</h1>
	<p>
		These are the core CSS variables used across the app. No component should hard-code hex values.
	</p>

	<h2 style="margin-top: 2rem;">Colors</h2>
	<div
		style="display:grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 16px;"
	>
		{#each colors as [name, value] (name)}
			<div class="card" style="padding: 16px;">
				<div class="swatch" style={`background:${value};`}></div>
				<div style="margin-top: 8px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;">
					<div>--{name}</div>
					<div style="color:var(--cf-ink-muted);">{value}</div>
				</div>
			</div>
		{/each}
	</div>

	<h2 style="margin-top: 2rem;">Radii & Shadows</h2>
	<div
		style="display:grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 16px;"
	>
		<div
			class="card"
			style="padding:16px; border-radius: var(--cf-radius-sm); box-shadow: var(--cf-shadow-sm);"
		>
			<strong>radius-sm + shadow-sm</strong>
		</div>
		<div
			class="card"
			style="padding:16px; border-radius: var(--cf-radius-md); box-shadow: var(--cf-shadow-md);"
		>
			<strong>radius-md + shadow-md</strong>
		</div>
		<div class="card brand-gradient" style="padding:16px; border-radius: var(--cf-radius-lg);">
			<strong>gradient + radius-lg</strong>
		</div>
	</div>

	<h2 style="margin-top: 2rem;">Typography</h2>
	<div class="card" style="padding: 16px;">
		<h1 style="margin: 0;">Heading H1</h1>
		<p style="margin-top: 8px;">
			Body text using Inter/system stack. This page is for visual verification only.
		</p>
	</div>
</section>

{@render children?.()}
