<script lang="ts">
	import Button from '$lib/ui/Button.svelte';
	import { browser } from '$app/environment';
	let { data, form } = $props();

	let step = $state<'idle' | 'pending' | 'approved' | 'error'>('idle');
	let deviceCode = $state<string | null>(null);
	let userCode = $state<string | null>(null);
	let verificationUri = $state<string | null>(null);
	let expiresIn = $state<number | null>(null);
	let interval = $state<number>(5);
	let errorMsg = $state<string | null>(null);

	let pollTimer: any;

	function startPolling() {
		clearInterval(pollTimer);
		pollTimer = setInterval(async () => {
			if (!deviceCode) return;
			const fd = new FormData();
			fd.set('device_code', deviceCode);
			const r = await fetch('?/poll', {
				method: 'POST',
				body: fd,
				headers: { accept: 'application/json' }
			});
			const res = await r.json();
			if (res?.slowDown) {
				interval = Math.min(interval + 5, 60);
				startPolling();
				return;
			}
			if (res?.pending) return;
			if (res?.done) {
				step = 'approved';
				clearInterval(pollTimer);
				return;
			}
			if (res?.error) {
				errorMsg = res.error;
				step = 'error';
				clearInterval(pollTimer);
			}
		}, interval * 1000);
	}
</script>

<section class="section container" style="max-width:720px;">
	<h1>Approve a Device</h1>

	{#if step === 'idle'}
		<form method="POST" action="?/init" class="stack-4">
			<div class="stack-2 grid gap-2">
				<label for="name">Device name</label>
				<input id="name" name="name" placeholder="My Laptop" />
			</div>
			<div class="stack-2 grid gap-2">
				<label for="platform">Platform</label>
				<input id="platform" name="platform" placeholder="macOS" />
			</div>
			<div class="stack-2 grid gap-2">
				<label for="fingerprint">Fingerprint (optional)</label>
				<input id="fingerprint" name="fingerprint" placeholder="auto-generated" />
			</div>
			<Button type="submit">Start</Button>
		</form>

		{#if form?.init}
			<!-- Values will be applied via reactive block on the client -->
		{/if}
	{/if}

	{#if step === 'pending'}
		<div class="stack-4">
			<p>Use another signed-in session to approve this device.</p>
			<div class="card" style="padding:16px;">
				<div class="text-subtle">User code</div>
				<div
					style="font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 1.25rem;"
				>
					{userCode}
				</div>
			</div>
			<p>
				<a class="nav-link" href={verificationUri!} rel="noreferrer" target="_blank"
					>Open approval page</a
				>
			</p>
			<small class="text-subtle">Polling every {interval}s. Expires in ~{expiresIn}s.</small>
		</div>
	{/if}

	{#if step === 'approved'}
		<p>Device approved and session established.</p>
		<p><a class="nav-link" href="/">Continue to app</a></p>
	{/if}

	{#if step === 'error'}
		<p style="color:crimson;">{errorMsg}</p>
		<p class="text-subtle">Restart the device authorization if needed.</p>
	{/if}
</section>

<!-- Apply init result after client hydration to avoid SSR relative fetch error -->
{#if browser && form?.init && step === 'idle'}
	{@const init = form.init as any}
	{@html (() => {
		deviceCode = init.device_code;
		userCode = init.user_code;
		verificationUri = init.verification_uri;
		expiresIn = init.expires_in;
		interval = init.interval ?? 5;
		step = 'pending';
		startPolling();
		return '';
	})()}
{/if}
