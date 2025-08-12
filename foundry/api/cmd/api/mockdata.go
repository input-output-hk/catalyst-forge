package main

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
)

// MockDataCmd populates the database with mock releases, deployments, and events.
type MockDataCmd struct {
	// Comma-separated project names
	Projects string `kong:"help='Comma-separated project names',default='alpha,beta,gamma'"`
	// Comma-separated branch names
	Branches string `kong:"help='Comma-separated branch names',default='main,develop,release/1.x'"`
	// Number of releases to create per project
	Releases int `kong:"help='Number of releases per project',default=100"`
	// Number of deployments to create per release
	Deployments int `kong:"help='Number of deployments per release',default=3"`
	// Number of events to create per deployment
	Events int `kong:"help='Number of events per deployment',default=6"`
}

func (c *MockDataCmd) Run() error {
	cfg := configFromEnv()
	// Build a basic logger; reuse config's logger helper for consistency
	logger, _ := cfg.GetLogger()
	db, err := openDB(cfg, logger)
	if err != nil {
		return err
	}

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("mockdata: run migrations: %w", err)
	}

	// Initialize repositories / services used for creation
	releaseRepo := repository.NewReleaseRepository(db)
	deploymentRepo := repository.NewDeploymentRepository(db)
	aliasRepo := repository.NewAliasRepository(db)
	counterRepo := repository.NewIDCounterRepository(db)
	eventRepo := repository.NewEventRepository(db)

	releaseSvc := service.NewReleaseService(releaseRepo, aliasRepo, counterRepo, deploymentRepo)

	projects := splitAndTrim(c.Projects)
	branches := splitAndTrim(c.Branches)
	if len(projects) == 0 || len(branches) == 0 {
		return fmt.Errorf("projects and branches must not be empty")
	}

	// Seed RNG
	mrand.Seed(time.Now().UnixNano())

	now := time.Now()
	for _, project := range projects {
		// Track latest release per project and per branch for aliasing
		latestProjectReleaseID := ""
		latestByBranch := map[string]string{}

		for i := 0; i < c.Releases; i++ {
			// Stagger creation times in the past for nicer ordering
			created := now.Add(-time.Duration((i+1)*mrand.Intn(60)) * time.Minute)
			branch := branches[i%len(branches)]
			release := &models.Release{
				SourceRepo:   fmt.Sprintf("https://github.com/example/%s", project),
				SourceCommit: randomHex(20),
				SourceBranch: branch,
				Project:      project,
				ProjectPath:  fmt.Sprintf("%s/service-%d", project, (i%5)+1),
				Created:      created,
				Bundle:       fmt.Sprintf("{\"project\":\"%s\",\"i\":%d,\"branch\":\"%s\"}", project, i, branch),
			}

			if err := releaseSvc.CreateRelease(defaultCtx(), release); err != nil {
				return fmt.Errorf("create release: %w", err)
			}

			// Track latest ids for aliasing
			latestProjectReleaseID = release.ID
			latestByBranch[branch] = release.ID

			// Create deployments for this release
			for d := 0; d < c.Deployments; d++ {
				ts := created.Add(time.Duration(d+1) * time.Minute)
				status := randomStatus()
				attempts := mrand.Intn(3)
				dep := &models.ReleaseDeployment{
					ID:        fmt.Sprintf("%s-%d", release.ID, ts.UnixNano()),
					ReleaseID: release.ID,
					Timestamp: ts,
					Status:    status,
					Reason:    reasonForStatus(status),
					Attempts:  attempts,
				}
				if err := deploymentRepo.Create(defaultCtx(), dep); err != nil {
					return fmt.Errorf("create deployment: %w", err)
				}

				// Add events for this deployment
				eventTs := ts
				for e := 0; e < c.Events; e++ {
					name, msg := eventForIndex(e, status)
					eventTs = eventTs.Add(time.Duration(10+mrand.Intn(50)) * time.Second)
					ev := &models.DeploymentEvent{
						DeploymentID: dep.ID,
						Name:         name,
						Message:      msg,
						Timestamp:    eventTs,
					}
					if err := eventRepo.AddEvent(defaultCtx(), ev); err != nil {
						return fmt.Errorf("add event: %w", err)
					}
				}
			}

			// Create project-level aliases pointing to most recent release
			if latestProjectReleaseID != "" {
				// project-latest and project-stable
				_ = releaseSvc.CreateReleaseAlias(defaultCtx(), fmt.Sprintf("%s-latest", sanitizeAliasPart(project)), latestProjectReleaseID)
				_ = releaseSvc.CreateReleaseAlias(defaultCtx(), fmt.Sprintf("%s-stable", sanitizeAliasPart(project)), latestProjectReleaseID)
			}
			// Create branch-scoped latest aliases per project
			for br, rid := range latestByBranch {
				alias := fmt.Sprintf("%s-%s-latest", sanitizeAliasPart(project), sanitizeAliasPart(br))
				_ = releaseSvc.CreateReleaseAlias(defaultCtx(), alias, rid)
			}
		}
	}

	fmt.Printf("Mock data generated successfully for %d project(s).\n", len(projects))
	return nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// sanitizeAliasPart converts a string into a safe alias fragment
// Replace spaces and slashes; lowercase for consistency.
func sanitizeAliasPart(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-")
	return replacer.Replace(s)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		// fallback to math/rand if crypto fails
		for i := range b {
			b[i] = byte(97 + mrand.Intn(26))
		}
	}
	return hex.EncodeToString(b)
}

func randomStatus() models.DeploymentStatus {
	switch mrand.Intn(4) {
	case 0:
		return models.DeploymentStatusPending
	case 1:
		return models.DeploymentStatusRunning
	case 2:
		return models.DeploymentStatusSucceeded
	default:
		return models.DeploymentStatusFailed
	}
}

func reasonForStatus(s models.DeploymentStatus) string {
	switch s {
	case models.DeploymentStatusSucceeded:
		return ""
	case models.DeploymentStatusFailed:
		return "Non-zero exit code"
	case models.DeploymentStatusRunning:
		return "In progress"
	default:
		return "Queued"
	}
}

func eventForIndex(i int, final models.DeploymentStatus) (string, string) {
	switch i {
	case 0:
		return "queued", "Deployment queued"
	case 1:
		return "start", "Deployment started"
	default:
		// penultimate and last events convey final state
		if i >= 4 {
			if final == models.DeploymentStatusSucceeded {
				return "complete", "Deployment succeeded"
			}
			if final == models.DeploymentStatusFailed {
				return "error", "Deployment failed"
			}
		}
		return "log", fmt.Sprintf("Processing step %d", i)
	}
}

func defaultCtx() context.Context { return context.Background() }
