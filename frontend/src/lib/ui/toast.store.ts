import { writable } from 'svelte/store';

export type Toast = { id: number; text: string };

export const toasts = writable<Toast[]>([]);

let nextId = 1;

export function show(text: string, ms = 2500) {
	const id = nextId++;
	toasts.update((arr) => [...arr, { id, text }]);
	setTimeout(() => {
		toasts.update((arr) => arr.filter((t) => t.id !== id));
	}, ms);
}
