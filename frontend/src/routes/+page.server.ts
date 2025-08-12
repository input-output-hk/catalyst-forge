import type { PageServerLoad } from './$types';
import { apiClient } from '$lib/api/client';

type Release = {
	id: string;
	project?: string;
	created_at?: string;
};

type Deployment = {
	id: string;
	release_id: string;
	status: 'pending' | 'running' | 'succeeded' | 'failed';
	timestamp: string;
};

export const load: PageServerLoad = async ({ parent }) => {
	const { apiBaseUrl } = await parent();
	const client = apiClient(apiBaseUrl);

	try {
		const { data, error, response } = await client.GET('/releases');

		if (response?.status === 401 || response?.status === 403) {
			const now = new Date();
			const releases: Release[] = [
				{ id: 'r-001', project: 'example-project', created_at: now.toISOString() },
				{
					id: 'r-000',
					project: 'another',
					created_at: new Date(now.getTime() - 864e5).toISOString()
				}
			];
			const deployments: Deployment[] = [
				{
					id: 'd-100',
					release_id: 'r-001',
					status: 'succeeded',
					timestamp: new Date(now.getTime() - 10 * 60 * 1000).toISOString()
				},
				{
					id: 'd-099',
					release_id: 'r-000',
					status: 'failed',
					timestamp: new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString()
				}
			];
			return { mocked: true, releases, deployments };
		}

		if (error)
			return { error: 'Failed to load data', releases: [], deployments: [], mocked: false };

		const releases = (data ?? []) as Release[];

		// Fetch latest deployment for up to 8 recent releases
		const top = releases.slice(0, 8);
		const results = await Promise.all(
			top.map(async (r) => {
				const { data: depData } = await client.GET('/release/{id}/deploy/latest', {
					params: { path: { id: r.id } }
				});
				if (!depData) return null;
				const d = depData as any;
				const item: Deployment = {
					id: d.id,
					release_id: d.release_id ?? r.id,
					status: d.status,
					timestamp: d.timestamp ?? d.created_at ?? new Date().toISOString()
				};
				return item;
			})
		);

		const deployments = results.filter(Boolean) as Deployment[];

		return { mocked: false, releases, deployments };
	} catch {
		return { error: 'Failed to load data', releases: [], deployments: [], mocked: false };
	}
};
