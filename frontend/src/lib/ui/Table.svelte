<script lang="ts">
	type Row = Record<string, unknown>;
	export type SimpleColumn<RowType extends Row> = {
		id: string;
		header: string;
		accessor?: (row: RowType) => unknown;
		cell?: (row: RowType) => string | number; // rendered as trusted HTML when string
		class?: string;
		headerClass?: string;
	};
	let { columns, data } = $props() as { columns: SimpleColumn<Row>[]; data: Row[] };
</script>

<div class="cf-table">
	<table>
		<thead>
			<tr>
				{#each columns as col (col.id)}
					<th class={col.headerClass}>{col.header}</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each data as row, i (i)}
				<tr>
					{#each columns as col (col.id)}
						<td class={col.class}>
							{#if col.cell}
								{@html col.cell(row as any) as unknown as string}
							{:else if col.accessor}
								{col.accessor(row as any)}
							{:else}{/if}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<style>
	.cf-table {
		border: 1px solid var(--cf-border-subtle);
		border-radius: var(--cf-radius-md);
		overflow: hidden;
		background: var(--cf-surface);
	}
	.cf-table table {
		width: 100%;
		border-collapse: separate;
		border-spacing: 0;
	}
	.cf-table th,
	.cf-table td {
		text-align: left;
		padding: 10px 12px;
		border-bottom: 1px solid var(--cf-border-subtle);
	}
	.cf-table thead th {
		color: var(--cf-text-subtle);
		font-weight: 600;
		position: sticky;
		top: 0;
		background: var(--cf-surface);
		z-index: 1;
	}
	.cf-table tbody tr:nth-child(odd) {
		background: var(--cf-surface);
	}
	.cf-table tbody tr:nth-child(even) {
		background: var(--cf-surface-soft);
	}
	.cf-table tbody tr:hover {
		background: color-mix(in oklab, var(--cf-blue-500) 6%, white);
	}

	/* Status pill */
	.pill {
		display: inline-block;
		padding: 2px 8px;
		border-radius: 9999px;
		font-size: 0.875rem;
		line-height: 1.4;
		border: 1px solid var(--cf-border-subtle);
		background: var(--cf-surface);
	}
	.pill--active {
		color: var(--cf-blue-600);
		background: color-mix(in oklab, var(--cf-blue-500) 12%, white);
		border-color: color-mix(in oklab, var(--cf-blue-500) 35%, white);
	}
	.pill--paused {
		color: var(--cf-text-subtle);
		background: color-mix(in oklab, var(--cf-ink) 4%, white);
	}
	.pill--archived {
		color: var(--cf-text-subtle);
		background: var(--cf-surface-soft);
		opacity: 0.95;
	}
</style>
