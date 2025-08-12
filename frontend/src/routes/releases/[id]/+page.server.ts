import type { PageServerLoad } from './$types';
import { apiClient } from '$lib/api/client';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, parent }) => {
	const { apiBaseUrl } = await parent();
	const client = apiClient(apiBaseUrl);
	const id = params.id;

	const releaseRes = await client.GET('/release/{id}', { params: { path: { id } } });
	if (releaseRes.response.status === 401 || releaseRes.response.status === 403) {
		// Mock fallback for unauthenticated dev to validate table UI
		const mockRelease = {
			id,
			project: 'example-project',
			source_repo: 'github.com/org/repo',
			source_branch: 'main',
			source_commit: 'deadbeefcafebabe0123456789abcdef01234567'
		} as any;
		const mockDeployments = [
			{ id: 'd-001', status: 'succeeded', timestamp: new Date().toISOString(), reason: 'OK' },
			{
				id: 'd-000',
				status: 'failed',
				timestamp: new Date(Date.now() - 36e5).toISOString(),
				reason: 'Image pull error'
			},
			{
				id: 'd-00x',
				status: 'running',
				timestamp: new Date(Date.now() - 18e5).toISOString(),
				reason: 'Rolling update'
			}
		];
		return { release: mockRelease, deployments: mockDeployments, mocked: true };
	}

	if (!releaseRes.response.ok) throw error(releaseRes.response.status, 'Failed to load release');

	const deploymentsRes = await client.GET('/release/{id}/deployments', {
		params: { path: { id } }
	});
	if (deploymentsRes.response.status === 401 || deploymentsRes.response.status === 403) {
		return { release: releaseRes.data, deployments: [], mocked: true };
	}
	if (!deploymentsRes.response.ok)
		throw error(deploymentsRes.response.status, 'Failed to load deployments');

	const deployments = deploymentsRes.data ?? [];
	return { release: releaseRes.data, deployments };
};
