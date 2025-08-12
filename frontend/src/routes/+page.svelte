<script lang="ts">
	import Card from '$lib/ui/Card.svelte';
	import Table, { type SimpleColumn } from '$lib/ui/Table.svelte';

	// Data from +page.server.ts
	let { data } = $props();

	type Deployment = {
		id: string;
		release_id: string;
		status: 'pending' | 'running' | 'succeeded' | 'failed';
		timestamp: string;
	};

	type Row = { id: string; project?: string; status: string; time: string };

	let mocked = data.mocked as boolean;
	let releases = data.releases as Array<{ id: string; project?: string; created_at?: string }>;
	let deployments = data.deployments as Deployment[];

	// KPI metrics
	let total = $state(releases.length);
	let latestSucceeded = $state(deployments.filter((d) => d.status === 'succeeded').length);
	let latestFailed = $state(deployments.filter((d) => d.status === 'failed').length);
	$effect(() => {
		total = releases.length;
		latestSucceeded = deployments.filter((d) => d.status === 'succeeded').length;
		latestFailed = deployments.filter((d) => d.status === 'failed').length;
	});

	// Recent activity table rows
	let rows = $state<Row[]>([]);
	$effect(() => {
		rows = deployments
			.slice()
			.sort((a, b) => b.timestamp.localeCompare(a.timestamp))
			.map<Row>((d) => ({
				id: d.release_id,
				project: releases.find((r) => r.id === d.release_id)?.project,
				status: d.status,
				time: d.timestamp
			}));
	});

	const columns: SimpleColumn<Row>[] = [
		{ id: 'id', header: 'Release', accessor: (r) => r.id },
		{ id: 'project', header: 'Project', accessor: (r) => r.project ?? '' },
		{
			id: 'status',
			header: 'Status',
			cell: (r) => {
				const s = (r as Row).status;
				const cls =
					s === 'succeeded'
						? 'pill pill--active'
						: s === 'failed'
							? 'pill pill--archived'
							: 'pill pill--paused';
				return `<span class="${cls}">${s}</span>` as unknown as any;
			}
		},
		{ id: 'time', header: 'Time', accessor: (r) => r.time }
	];
</script>

<section class="hero-band section">
	<div class="stack-2 container">
		<h1 style="margin:0;">Welcome to Catalyst Forge</h1>
		<p class="text-subtle" style="margin:0;">Your projects at a glance</p>
	</div>
</section>

<section class="section cards-grid container">
	<Card title="Releases" variant="stat">{total}</Card>
	<Card title="Succeeded" variant="stat">{latestSucceeded}</Card>
	<Card title="Failed" variant="stat">{latestFailed}</Card>
</section>

<section class="section container">
	<h2>Recent Activity</h2>
	{#if mocked}
		<p class="text-subtle">
			Showing mocked data (401/403 from API). TODO: switch to real API when auth is wired.
		</p>
	{/if}
	<Table {columns} data={rows} />
</section>

<style>
	.text-subtle {
		color: var(--cf-text-subtle);
	}
	.cards-grid {
		display: grid;
		gap: var(--space-4);
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
	}
	section.brand-gradient {
		padding-block: var(--space-7);
	}
</style>
