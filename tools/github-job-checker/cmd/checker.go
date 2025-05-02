package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// Checker holds configuration for checking GitHub actions.
type Checker struct {
	Owner         string
	Repo          string
	Ref           string
	CheckInterval time.Duration
	Timeout       time.Duration
	Client        GitHubClient
}

// Run executes the check process.
func (c *Checker) Run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	ticker := time.NewTicker(c.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout of %v reached. GitHub jobs did not finish in time", c.Timeout)
		case <-ticker.C:
			results, err := c.Client.FetchCheckRunsForRef(ctx, c.Owner, c.Repo, c.Ref)
			if err != nil {
				return err
			}

			if results.GetTotal() == 0 {
				log.Print("No GitHub jobs configured for this commit.")
				return nil
			}

			var anyFailure, anyPending int

			for _, checkRun := range results.CheckRuns {
				status := checkRun.GetStatus()
				conclusion := checkRun.GetConclusion()

				if status == "completed" && conclusion != "success" {
					anyFailure++
				}

				if status == "in_progress" || status == "queued" {
					anyPending++
				}
			}

			log.Printf("Number of failed check runs: %d", anyFailure)
			log.Printf("Number of pending check runs: %d", anyPending)

			if anyFailure > 0 {
				return errors.New("one or more GitHub jobs failed")
			}

			if anyPending == 0 {
				log.Print("All GitHub jobs succeeded.")
				return nil
			}

			log.Printf("GitHub jobs are still running. Waiting for %v before rechecking.", c.CheckInterval)
		}
	}
}
