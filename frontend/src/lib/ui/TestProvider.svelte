<script lang="ts">
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	let { children } = $props();
	const qc = new QueryClient();
</script>

<QueryClientProvider client={qc}>
	{@render children?.()}
</QueryClientProvider>
