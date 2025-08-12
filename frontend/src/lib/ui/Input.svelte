<script lang="ts">
	let {
		value = '',
		placeholder = '',
		type = 'text',
		name,
		disabled = false,
		required = false,
		id = undefined as string | undefined,
		ariaInvalid = false as boolean
	} = $props();
</script>

<input
	class="cf-input"
	bind:value
	{placeholder}
	{type}
	{name}
	{id}
	{disabled}
	{required}
	aria-invalid={ariaInvalid}
/>

<style>
	.cf-input {
		width: 100%;
		padding: 0.5rem 0.75rem;
		border-radius: var(--cf-radius-sm);
		border: 1px solid color-mix(in oklab, var(--cf-ink) 12%, transparent);
		background: var(--cf-bg);
		color: var(--cf-ink);
	}
	.cf-input:focus-visible {
		outline: none;
		box-shadow: 0 0 0 3px color-mix(in oklab, var(--cf-blue-500) 35%, white);
		border-color: var(--cf-blue-500);
	}

	/* Invalid state (only after user input or submit). Avoid showing on initial empty state */
	.cf-input:invalid:not(:placeholder-shown),
	.cf-input:user-invalid,
	.cf-input[aria-invalid='true'] {
		border-color: var(--cf-danger);
		box-shadow: 0 0 0 3px color-mix(in oklab, var(--cf-danger) 25%, white);
	}
</style>
