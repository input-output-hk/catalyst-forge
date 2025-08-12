<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { mockClient } from '$lib/mocks/client';

	const projects = createQuery({
		queryKey: ['projects'],
		queryFn: () =>
			mockClient.getProjects().then((r) => {
				if (r.error) throw new Error(r.error);
				return r.data ?? [];
			})
	});

	const flaky = createQuery({
		queryKey: ['flaky'],
		queryFn: () =>
			mockClient.getFlaky().then((r) => {
				if (r.error) throw new Error(r.error);
				return r.data;
			})
	});
</script>

<section class="container" style="padding:24px 0;">
	<h1>Mock Query Demo</h1>

	<h2 style="margin-top:1rem;">Projects</h2>
	{#if $projects.isLoading}
		<p>Loading projects…</p>
	{:else if $projects.isError}
		<p style="color:crimson;">Error: {$projects.error?.message}</p>
	{:else}
		<ul>
			{#each $projects.data as p (p.id)}
				<li>{p.name} — {p.status}</li>
			{/each}
		</ul>
	{/if}

	<h2 style="margin-top:1rem;">Flaky Example</h2>
	{#if $flaky.isLoading}
		<p>Loading flaky…</p>
	{:else if $flaky.isError}
		<p style="color:crimson;">Error (expected sometimes): {$flaky.error?.message}</p>
	{:else}
		<p>Success: {JSON.stringify($flaky.data)}</p>
	{/if}
</section>
