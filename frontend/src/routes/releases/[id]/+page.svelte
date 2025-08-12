<script lang="ts">
	import Table, { type SimpleColumn } from '$lib/ui/Table.svelte';
	type Deployment = {
		id?: string;
		status?: string;
		timestamp?: string;
		reason?: string;
	};
	let { data } = $props();
	const columns: SimpleColumn<Deployment>[] = [
		{ id: 'id', header: 'ID', accessor: (r) => r.id ?? '' },
		{ id: 'status', header: 'Status', accessor: (r) => r.status ?? '' },
		{ id: 'timestamp', header: 'Time', accessor: (r) => r.timestamp ?? '' },
		{ id: 'reason', header: 'Reason', accessor: (r) => r.reason ?? '' }
	];
</script>

<section class="section container">
	<h1>Release {data.release?.id}</h1>
	<div class="stack-3" style="margin-bottom: 16px;">
		<div class="text-subtle">Project</div>
		<div>{data.release?.project}</div>
		<div class="text-subtle">Source</div>
		<div>
			{data.release?.source_repo}#{data.release?.source_branch}@{(
				data.release?.source_commit ?? ''
			).slice(0, 7)}
		</div>
	</div>

	<h2>Deployments</h2>
	{#if data.mocked}
		<p class="text-subtle">
			Showing mocked data (401/403 from API). TODO: switch to real API when auth is wired.
		</p>
	{/if}
	{#if !data.deployments?.length}
		<p class="text-subtle">No deployments found.</p>
	{:else}
		<Table {columns} data={data.deployments} />
	{/if}
</section>
