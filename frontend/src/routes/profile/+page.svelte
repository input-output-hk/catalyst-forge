<script lang="ts">
	let { data } = $props();
	let showRaw = $state(false);

	function decodeJwt(token: string): any | null {
		if (!token) return null;
		try {
			const payload = token.split('.')[1];
			const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
			return JSON.parse(json);
		} catch {
			return null;
		}
	}

	const claims = decodeJwt(data.token);
</script>

<section class="section container" style="max-width:720px;">
	<h1>Profile</h1>
	{#if data.mocked}
		<p class="text-subtle">
			Showing mocked session list. TODO: replace with real `/tokens/*` endpoints once auth is wired.
		</p>
	{/if}

	<div class="stack-4">
		<div>
			<h2>Identity</h2>
			{#if claims}
				<pre
					style="white-space:pre-wrap; background:var(--cf-bg-soft); padding: var(--space-3); border-radius: var(--radius-2);">{JSON.stringify(
						claims,
						null,
						2
					)}</pre>
			{:else}
				<p class="text-subtle">No access token available. Sign in to view claims.</p>
			{/if}
			<p><a class="nav-link" href={data.jwksUrl}>JWKS</a></p>
		</div>

		<div>
			<h2>Sessions</h2>
			<ul class="stack-2">
				{#each data.sessions as s}
					<li class="card" style="padding: var(--space-3);">
						<div style="display:flex; justify-content: space-between; align-items: center;">
							<div>
								<div><strong>{s.user_agent}</strong></div>
								<div class="text-subtle">{s.created_at}</div>
							</div>
							<form method="post" action="?/revoke">
								<input type="hidden" name="session_id" value={s.id} />
								<button class="nav-link" type="submit">Revoke</button>
							</form>
						</div>
					</li>
				{/each}
			</ul>
			<form method="post" action="?/refresh">
				<button class="nav-link" type="submit">Refresh token</button>
			</form>
		</div>
	</div>
</section>
