import { render, fireEvent } from '@testing-library/svelte';
// jest-dom matchers are extended via vitest-setup-client.ts
import Button from '../Button.svelte';
import { describe, it, expect, vi } from 'vitest';

describe('Button (browser)', () => {
	it('renders slot content', () => {
		const { getByRole } = render(Button, { props: { children: () => 'Click' } });
		expect(getByRole('button')).toBeInTheDocument();
	});

	it('emits click', async () => {
		const { getAllByRole } = render(Button, { props: { children: () => 'Do' } });
		const [btn] = getAllByRole('button');
		const clicked = vi.fn();
		btn.addEventListener('click', clicked);
		await fireEvent.click(btn);
		expect(clicked).toHaveBeenCalled();
	});
});
