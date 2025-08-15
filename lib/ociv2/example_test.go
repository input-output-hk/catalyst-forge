package ociv2_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/ociv2"
)

func ExampleClient_PushReleaseBundle() {
	// Create a client with options
	client, err := ociv2.New(ociv2.ClientOptions{
		Timeout:   2 * time.Minute,
		UserAgent: "forge-oci/1.0",
		Cosign: ociv2.CosignOpts{
			Enable: true,
			Identity: &ociv2.OIDCIdentity{
				Issuer:  "https://token.actions.githubusercontent.com",
				Subject: "repo:input-output-hk/catalyst-forge:ref:refs/heads/main",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a Release Bundle JSON
	releaseJSON := []byte(`{
		"releaseKey": "foundry-operator-007",
		"version": "1.0.0",
		"components": ["api", "worker"]
	}`)

	// Push the Release Bundle
	ctx := context.Background()
	desc, err := client.PushReleaseBundle(
		ctx,
		"oci://registry.example.com/forge/releases/foundry-operator:v1.0.0",
		releaseJSON,
		ociv2.Annotations{
			ociv2.AnnForgeProject: "foundry-operator",
			ociv2.AnnForgeEnv:     "production",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Pushed release bundle to: %s\n", desc.Ref)
	// Output would be something like:
	// Pushed release bundle to: oci://registry.example.com/forge/releases/foundry-operator@sha256:abc123...
}

func ExampleClient_PullReleaseBundle() {
	// Create a client
	client, err := ociv2.New(ociv2.ClientOptions{
		Timeout: 2 * time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Pull a Release Bundle by digest
	ctx := context.Background()
	data, desc, err := client.PullReleaseBundle(
		ctx,
		"oci://registry.example.com/forge/releases/foundry-operator@sha256:abc123",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Pulled release bundle: %d bytes, media type: %s\n", len(data), desc.MediaType)
	// Output would be:
	// Pulled release bundle: 256 bytes, media type: application/vnd.forge.release+json
}

func ExampleStaticAuth() {
	// Create a client with static authentication
	client, err := ociv2.New(ociv2.ClientOptions{
		Auth: &ociv2.StaticAuth{
			Username: "myuser",
			Password: "mypassword",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Use the client...
	_ = client
}

func ExampleGitHubAuth() {
	// Create a client with GitHub authentication for GHCR
	client, err := ociv2.New(ociv2.ClientOptions{
		Auth: &ociv2.GitHubAuth{
			Token: "ghp_yourtoken", // Or use GITHUB_TOKEN env var
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Push to GitHub Container Registry
	ctx := context.Background()
	releaseJSON := []byte(`{"version": "1.0.0"}`)
	
	desc, err := client.PushReleaseBundle(
		ctx,
		"oci://ghcr.io/myorg/myapp/releases:latest",
		releaseJSON,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Pushed to GHCR: %s\n", desc.Ref)
}

func ExampleAnnotations() {
	// Create annotations with builder pattern
	ann := ociv2.NewAnnotations().
		WithSource("https://github.com/example/repo", "abc123").
		WithForgeKind("release").
		WithForgeProject("my-project").
		WithForgeEnv("production").
		WithBuildInfo("build-123", "42", "https://ci.example.com/builds/42")

	// Use annotations when pushing
	client, _ := ociv2.New(ociv2.ClientOptions{})
	ctx := context.Background()
	
	_, err := client.PushReleaseBundle(
		ctx,
		"oci://registry.example.com/releases:latest",
		[]byte(`{}`),
		ann,
	)
	if err != nil {
		log.Fatal(err)
	}
}