import { render, fireEvent } from '@testing-library/svelte';
// jest-dom matchers are extended via vitest-setup-client.ts
import Input from '../Input.svelte';
import { describe, it, expect } from 'vitest';

describe('Input (browser)', () => {
	it('binds value and updates on input', async () => {
		const { getByRole } = render(Input, { props: { value: '', placeholder: 'Email' } });
		const input = getByRole('textbox') as HTMLInputElement;
		expect(input).toBeInTheDocument();
		await fireEvent.input(input, { target: { value: 'user@example.com' } });
		expect((getByRole('textbox') as HTMLInputElement).value).toBe('user@example.com');
	});
});
