<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { queryClient } from '$lib/query';
	import CommandPalette from '$lib/ui/CommandPalette.svelte';
	import { browser } from '$app/environment';

	let navOpen = $state(false);
	const links = [
		{ href: '/', label: 'Home' },
		{ href: '/releases', label: 'Releases' },
		{ href: '/profile', label: 'Profile' }
	];

	let { children, data } = $props();

	let paletteOpen = $state(false);
	function togglePalette() {
		paletteOpen = !paletteOpen;
	}
	$effect(() => {
		if (!browser) return;
		const onKey = (e: KeyboardEvent) => {
			if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
				e.preventDefault();
				togglePalette();
			}
		};
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<header class="site-header">
	<div class="header-inner container">
		<a href="/" class="brand">Catalyst Forge</a>
		<nav class={`nav ${navOpen ? 'open' : ''}`}>
			{#each links as link (link.href)}
				<a
					class="nav-link"
					href={link.href}
					aria-current={data?.pathname === link.href ? 'page' : undefined}>{link.label}</a
				>
			{/each}
			{#if data?.sessionEmail}
				<span class="nav-label">{data.sessionEmail}</span>
				<a class="nav-link" href="/logout">Logout</a>
			{:else}
				<a class="nav-link" href="/login">Login</a>
			{/if}
		</nav>
		<button class="menu" aria-label="Menu" onclick={() => (navOpen = !navOpen)}>☰</button>
	</div>
</header>

<main class="site-main section container">
	<QueryClientProvider client={queryClient}>
		{@render children?.()}
	</QueryClientProvider>
</main>

<CommandPalette open={paletteOpen} onClose={() => (paletteOpen = false)} />
