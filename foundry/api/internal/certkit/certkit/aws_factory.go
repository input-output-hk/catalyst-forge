package certkit

import (
	"context"

	provaws "github.com/input-output-hk/catalyst-forge/foundry/api/internal/certkit/provider/aws"
)

// BuildPCAFromRegion builds a PCA client using AWS default config chain for the given region.
func BuildPCAFromRegion(ctx context.Context, region string) (PCAClient, error) {
	cli, err := provaws.NewDefault(ctx, region)
	if err != nil {
		return nil, err
	}
	return awsPCAAdapter{inner: cli}, nil
}
