import { FoundryClient } from '../src';

async function main() {
  // Create client with bearer token
  const client = FoundryClient.withBearerToken(
    'https://api.foundry.example.com',
    'your-api-token'
  );

  // Or create from environment variables
  // const client = FoundryClient.fromEnv();

  try {
    // Perform health check
    const health = await client.healthCheck();
    console.log('Health check passed:', health);

    // Use the raw client for full API access
    const { data, error, response } = await client.raw.GET('/auth/users', {
      body: {} as any, // Type would be properly inferred from schema
    });

    if (error) {
      console.error('Error fetching users:', error);
      // Check specific error types
      if (FoundryClient.isUnauthorized({ response })) {
        console.error('Authentication failed - check your token');
      }
    } else {
      console.log('Users:', data);
    }

    // Create a release (example)
    const releaseResponse = await client.raw.POST('/release', {
      params: {
        query: { deploy: 'false' },
      },
      body: {
        bundle: 'my-bundle',
        project: 'my-project',
        project_path: 'path/to/project',
        source_commit: 'abc123',
        source_repo: 'github.com/myorg/myrepo',
      },
    });

    if (releaseResponse.error) {
      console.error('Failed to create release:', releaseResponse.error);
    } else {
      console.log('Created release:', releaseResponse.data);
    }

  } catch (err) {
    console.error('Unexpected error:', err);
  }
}

// Run if executed directly
if (require.main === module) {
  main().catch(console.error);
}