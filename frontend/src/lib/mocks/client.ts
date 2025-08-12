type Result<T> = Promise<{ data?: T; error?: string }>;

export interface MockProject {
	id: string;
	name: string;
	status: 'active' | 'paused' | 'archived';
}

export const mockClient = {
	async getProjects(): Result<MockProject[]> {
		await new Promise((r) => setTimeout(r, 250));
		return {
			data: [
				{ id: 'p1', name: 'Bootstrap Catalyst', status: 'active' },
				{ id: 'p2', name: 'Governance Explorer', status: 'paused' },
				{ id: 'p3', name: 'Analytics Pipeline', status: 'active' }
			]
		};
	},
	async getProjectsMany(count = 60): Result<MockProject[]> {
		await new Promise((r) => setTimeout(r, 150));
		const names = ['Alpha', 'Beta', 'Gamma'];
		const statuses: MockProject['status'][] = ['active', 'paused', 'archived'];
		const data: MockProject[] = Array.from({ length: count }).map((_, i) => {
			const idx = i + 1;
			const name = `${names[i % names.length]} ${String(idx).padStart(2, '0')}`;
			return { id: `p${idx}`, name, status: statuses[i % statuses.length] };
		});
		return { data };
	},
	async getFlaky(): Result<{ ok: boolean }> {
		await new Promise((r) => setTimeout(r, 250));
		if (Math.random() < 0.5) return { error: 'random error' };
		return { data: { ok: true } };
	}
};
