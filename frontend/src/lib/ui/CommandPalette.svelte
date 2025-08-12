<script lang="ts">
	let { open = false, onClose } = $props<{ open: boolean; onClose: () => void }>();

	type Cmd = { label: string; hint?: string; href: string };
	const commands: Cmd[] = [
		{ label: 'Dashboard', href: '/' },
		{ label: 'Releases', href: '/releases' },
		{ label: 'Profile', href: '/profile' },
		{ label: 'Approve Device', href: '/auth/device/approve' }
	];

	let query = $state('');
	let activeIndex = $state(0);
	let inputEl: HTMLInputElement | null = null;

	function filtered(): Cmd[] {
		const q = query.trim().toLowerCase();
		if (!q) return commands;
		return commands.filter((c) => c.label.toLowerCase().includes(q));
	}

	function onKey(e: KeyboardEvent) {
		if (!open) return;
		const items = filtered();
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			activeIndex = Math.min(activeIndex + 1, items.length - 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			activeIndex = Math.max(activeIndex - 1, 0);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			const sel = items[activeIndex];
			if (sel) {
				onClose();
				window.location.assign(sel.href);
			}
		} else if (e.key === 'Escape') {
			e.preventDefault();
			onClose();
		}
	}

	$effect(() => {
		open;
		if (open) {
			queueMicrotask(() => inputEl?.focus());
		}
	});
</script>

{#if open}
	<div
		class="cp-backdrop"
		role="presentation"
		onclick={(e) => e.target === e.currentTarget && onClose()}
		onkeydown={onKey}
	>
		<div class="cp-panel" role="dialog" aria-modal="true" aria-label="Command palette">
			<input
				bind:this={inputEl}
				class="cp-input"
				placeholder="Type a command…"
				value={query}
				oninput={(e) => (query = (e.target as HTMLInputElement).value)}
			/>
			<div class="cp-list" role="listbox" aria-label="Commands">
				{#each filtered() as cmd, i}
					<a
						href={cmd.href}
						role="option"
						aria-selected={activeIndex === i}
						class={`cp-item ${activeIndex === i ? 'active' : ''}`}
						tabindex={activeIndex === i ? 0 : -1}
						onfocus={() => (activeIndex = i)}>{cmd.label}<span class="cp-hint">{cmd.hint}</span></a
					>
				{/each}
				{#if filtered().length === 0}
					<div class="cp-empty">No results</div>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.cp-backdrop {
		position: fixed;
		inset: 0;
		background: color-mix(in oklab, black 50%, transparent);
		display: grid;
		place-items: start center;
		padding-top: 15vh;
		z-index: 1000;
	}
	.cp-panel {
		width: min(720px, 92vw);
		background: var(--cf-surface, white);
		border: 1px solid var(--cf-border, #e5e7eb);
		border-radius: var(--radius-3, 12px);
		box-shadow: var(--shadow-3, 0 10px 30px rgba(0, 0, 0, 0.12));
		overflow: hidden;
	}
	.cp-input {
		width: 100%;
		padding: var(--space-3) var(--space-4);
		font-size: 16px;
		border: 0;
		border-bottom: 1px solid var(--cf-border, #e5e7eb);
		outline: none;
	}
	.cp-list {
		max-height: 360px;
		overflow: auto;
	}
	.cp-item {
		display: flex;
		justify-content: space-between;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-4);
		text-decoration: none;
		color: inherit;
	}
	.cp-item:hover,
	.cp-item.active {
		background: var(--cf-bg-soft, #f6f7fb);
	}
	.cp-hint {
		color: var(--cf-text-subtle);
		font-size: 12px;
	}
	.cp-empty {
		padding: var(--space-4);
		color: var(--cf-text-subtle);
	}
</style>
