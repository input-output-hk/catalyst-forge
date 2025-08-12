import type { PageServerLoad } from './$types';
import { apiClient } from '$lib/api/client';

type Release = {
	id: string;
	project?: string;
	source_repo?: string;
	source_branch?: string;
	source_commit?: string;
	created_at?: string;
};

export const load: PageServerLoad = async ({ parent }) => {
	const { apiBaseUrl } = await parent();
	const client = apiClient(apiBaseUrl);

	try {
		// Always render shell; client will fetch with withAuth to attach AT
		return { releases: [], mocked: false, needsClient: true };

		if (response?.status === 401 || response?.status === 403) {
			const now = new Date();
			const releases: Release[] = [
				{
					id: 'r-001',
					project: 'example-project',
					source_repo: 'github.com/org/repo',
					source_branch: 'main',
					source_commit: 'deadbeefcafebabe0123456789abcdef01234567',
					created_at: now.toISOString()
				},
				{
					id: 'r-000',
					project: 'another',
					source_repo: 'github.com/org/another',
					source_branch: 'release',
					source_commit: '0123456789abcdefdeadbeefcafebabe0123456',
					created_at: new Date(now.getTime() - 864e5).toISOString()
				}
			];
			return { releases, mocked: true };
		}

		if (error) {
			return { releases: [], mocked: false, error: 'Failed to load releases' };
		}

		// Unused path; we rely on client fetch for authed data
		return { releases: [], mocked: false, needsClient: true };
	} catch (e) {
		return { releases: [], mocked: false, error: 'Failed to load releases' };
	}
};
