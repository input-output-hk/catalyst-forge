<script lang="ts">
	import Table, { type SimpleColumn } from '$lib/ui/Table.svelte';
	import { page } from '$app/stores';
	import { browser } from '$app/environment';

	type Release = {
		id: string;
		project?: string;
		source_repo?: string;
		source_branch?: string;
		source_commit?: string;
		created_at?: string;
	};

	// Data provided by +page.server.ts
	let { data } = $props();
	let releases: Release[] = data.releases as Release[];
	let mocked = $state<boolean>(data.mocked ?? false);
let needsClient = $state<boolean>(data.needsClient ?? false);

	const columns: SimpleColumn<Release>[] = [
		{
			id: 'id',
			header: 'ID',
			cell: (r) =>
				`<a class="nav-link" href="/releases/${(r as Release).id}">${(r as Release).id}</a>`
		},
		{ id: 'project', header: 'Project', accessor: (r) => r.project ?? '' },
		{
			id: 'source',
			header: 'Source',
			accessor: (r) =>
				`${r.source_repo ?? ''}#${r.source_branch ?? ''}@${(r.source_commit ?? '').slice(0, 7)}`
		},
		{ id: 'created', header: 'Created', accessor: (r) => r.created_at ?? '' }
	];

	// UI state with URL persistence
	let q = $state(($page.url.searchParams.get('q') ?? '').toString());
	let density = $state<'comfortable' | 'compact'>(
		($page.url.searchParams.get('density') as any) ?? 'comfortable'
	);
	let sortKey = $state(($page.url.searchParams.get('sortKey') ?? 'created_at').toString());
	let sortDir = $state<'asc' | 'desc'>(($page.url.searchParams.get('sortDir') as any) ?? 'desc');

	function updateURL() {
		if (!browser) return;
		const url = new URL(window.location.href);
		if (q) url.searchParams.set('q', q);
		else url.searchParams.delete('q');
		url.searchParams.set('density', density);
		url.searchParams.set('sortKey', sortKey);
		url.searchParams.set('sortDir', sortDir);
		const next = url.toString();
		if (next !== window.location.href) {
			history.replaceState({}, '', url);
		}
	}

	// Only react to local state changes; do not depend on $page.url to avoid loops
	$effect(() => {
		q;
		density;
		sortKey;
		sortDir;
		updateURL();
	});

	function onHeaderClick(key: string) {
		if (sortKey === key) {
			sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		} else {
			sortKey = key;
			sortDir = 'asc';
		}
		// state changes trigger URL sync via $effect
	}

	function getSortedFiltered(data: Release[]): Release[] {
		let rows = data;
		if (q) {
			const needle = q.toLowerCase();
			rows = rows.filter((r) =>
				`${r.id} ${r.project ?? ''} ${r.source_repo ?? ''} ${r.source_branch ?? ''} ${r.source_commit ?? ''}`
					.toLowerCase()
					.includes(needle)
			);
		}
		rows = rows.slice().sort((a, b) => {
			const av = (a as any)[sortKey] ?? '';
			const bv = (b as any)[sortKey] ?? '';
			const c = String(av).localeCompare(String(bv));
			return sortDir === 'asc' ? c : -c;
		});
		return rows;
	}
</script>

<section class="section container">
	<h1>Releases</h1>
  {#if needsClient && browser}
    {@html (() => {
      (async () => {
        try {
          const { withAuth } = await import('$lib/auth/session');
          const authFetch = withAuth(fetch);
          const r = await authFetch('/releases');
          if (r.ok) {
            releases = await r.json();
            needsClient = false;
          } else {
            mocked = true;
          }
        } catch {
          mocked = true;
        }
      })();
      return '';
    })()}
  {/if}
	<div class="stack-4" style="margin-bottom: 12px;">
		<input placeholder="Search" bind:value={q} class="max-w-560" />
		<div class="grid gap-2" style="grid-auto-flow: column; width: fit-content;">
			<button
				class="nav-link"
				onclick={() => (density = density === 'comfortable' ? 'compact' : 'comfortable')}
				>Density: {density}</button
			>
		</div>
	</div>

	{#if data?.error && !mocked}
		<p style="color:crimson;">Error loading releases</p>
	{/if}
	{#if mocked}
		<p class="text-subtle">
			Showing mocked data (401/403 from API). TODO: switch to real API when auth is wired.
		</p>
	{/if}
	<Table {columns} data={getSortedFiltered(releases)} />
</section>
