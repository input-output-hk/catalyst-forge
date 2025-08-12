<script lang="ts">
	import Input from '$lib/ui/Input.svelte';
	import Button from '$lib/ui/Button.svelte';
	let email = $state('');
</script>

<section class="section container max-w-560">
	<h1>Login</h1>
	<p>Enter your email to receive a fake magic link.</p>
	<form method="GET" action="/auth/callback" class="stack-4">
		<div class="stack-2 grid gap-2">
			<label for="email">Email</label>
			<Input
				id="email"
				name="email"
				type="email"
				placeholder="you@example.com"
				bind:value={email}
				required
			/>
			<small class="text-subtle">We will never send a real email in this mock flow.</small>
		</div>
		<input type="hidden" name="token" value="mocktoken" />
		<Button type="submit">Send Link</Button>
	</form>
</section>
