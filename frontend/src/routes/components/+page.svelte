<script lang="ts">
	import Button from '$lib/ui/Button.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Card from '$lib/ui/Card.svelte';
	import Modal from '$lib/ui/Modal.svelte';
	import Drawer from '$lib/ui/Drawer.svelte';
	import ToastHost from '$lib/ui/ToastHost.svelte';
	import { show as showToast } from '$lib/ui/toast.store';

	let modalOpen = $state(false);
	let drawerOpen = $state(false);
	let email = $state('');
	let { children } = $props();

	$effect(() => {
		if (typeof window !== 'undefined') {
			const sp = new URLSearchParams(window.location.search);
			if (sp.get('openModal') === '1') modalOpen = true;
		}
	});
</script>

<ToastHost />

<section class="container" style="padding: 2rem 0; display: grid; gap: 24px;">
	<h1>UI Primitives</h1>

	<Card title="Buttons">
		<div style="display:flex; gap: 8px; flex-wrap: wrap;">
			<Button variant="primary">Primary</Button>
			<Button variant="secondary">Secondary</Button>
			<Button variant="ghost">Ghost</Button>
			<button
				class="cf-btn cf-btn--secondary"
				data-testid="open-modal"
				onclick={() => (modalOpen = true)}>Open Modal</button
			>
			<button
				class="cf-btn cf-btn--secondary"
				data-testid="open-drawer"
				onclick={() => (drawerOpen = true)}>Open Drawer</button
			>
			<button class="cf-btn cf-btn--ghost" onclick={() => showToast('Hello from Toast!')}
				>Show Toast</button
			>
		</div>
	</Card>

	<Card title="Inputs">
		<div style="display:grid; gap: 8px; max-width: 360px;">
			<label for="email">Email</label>
			<Input
				id="email"
				name="email"
				type="email"
				placeholder="you@example.com"
				bind:value={email}
			/>
			<div style="color:var(--cf-ink-muted)">Value: {email}</div>
		</div>
	</Card>

	<Modal bind:open={modalOpen}>
		<h3>Example Modal</h3>
		<p>Press Escape or click backdrop to close.</p>
		<div style="display:flex; gap: 8px; justify-content: end; margin-top: 12px;">
			<Button variant="secondary" on:click={() => (modalOpen = false)} autofocus>Close</Button>
			<Button on:click={() => (modalOpen = false)}>Confirm</Button>
		</div>
	</Modal>

	<Drawer bind:open={drawerOpen} side="right">
		<div style="display:flex; align-items:center; justify-content: space-between; gap: 12px;">
			<h3 style="margin: 0;">Example Drawer</h3>
			<Button variant="ghost" on:click={() => (drawerOpen = false)}>Close</Button>
		</div>
		<p>Press Escape or click outside to close.</p>
	</Drawer>
</section>

{@render children?.()}
