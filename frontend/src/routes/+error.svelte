<script lang="ts">
	type ErrorLike = { message?: string } | undefined;
	let { data, errors } = $props();
	const status: number = (data as { status?: number } | undefined)?.status ?? 500;
	const message: string =
		(errors as ErrorLike)?.message ??
		(data as { message?: string } | undefined)?.message ??
		'Something went wrong';
</script>

<section class="section container" style="max-width: 720px;">
	<h1>{status}</h1>
	<p class="text-subtle">{message}</p>
	<a class="cf-btn cf-btn--secondary focus-ring" href="/">Back home</a>
</section>
