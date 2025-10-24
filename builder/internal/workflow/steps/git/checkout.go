package git

import (
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

func init() {

	workflow.RegisterStep("git/checkout", newCHeckoutStep)
}

type CheckoutStep struct {
	destination string
}

func newCHeckoutStep(cfg sharedworkflow.StepConfig) (workflow.Step, error) {

	return &CheckoutStep{
		destination: cfg.Params["destination"],
	}, nil
}

// TODO: Context
func (s *CheckoutStep) Exec(ctx *workflow.WorkflowContext) (*workflow.StepResult, error) {
	var dest string
	if s.destination != "" {
		dest = s.destination
	} else {
		dest = ctx.Workspace
	}

	// Try cloning without authentication first (for public repos)
	log.Printf("Checking out %s to %s...", ctx.Source, dest)
	_, err := git.PlainClone(dest, false, &git.CloneOptions{
		URL:      ctx.Source,
		Progress: os.Stdout,
	})

	// If authentication is required and we have a token, retry with token
	// For Gitea private repos, you need to provide: username:token@host
	// Since we only have the token, we can't authenticate to private repos yet
	// TODO: Add GITEA_USERNAME env var for private repo support

	if err != nil {
		log.Printf("[git] Clone failed: %v", err)
		return &workflow.StepResult{
			Status: workflow.JobFailed,
		}, err
	}

	log.Printf("[git] Successfully cloned repository")

	return &workflow.StepResult{
		Status: workflow.StepSuccess,
	}, nil
}

func (s *CheckoutStep) Name() string {
	return "Git Checkout"
}
func (s *CheckoutStep) Description() string {
	return "Checkout Git repository"
}
