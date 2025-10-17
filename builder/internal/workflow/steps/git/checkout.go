package git

import (
	"fmt"
	"log"
	"os"

	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
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

	var auth *http.BasicAuth
	if token := os.Getenv("GIT_TOKEN"); token != "" {
		auth = &http.BasicAuth{
			Username: "topi-builder",
			Password: token,
		}
	} else {
		log.Printf("[git] no token found, attempting unauthenticated clone")
	}

	log.Printf("Checking out to %s...", dest)
	_, err := git.PlainClone(dest, false, &git.CloneOptions{
		URL:      ctx.Source,
		Progress: os.Stdout,
		Auth:     auth,
	})

	if err != nil {
		fmt.Println(fmt.Errorf("err:%s", err))
		return &workflow.StepResult{
			Status: workflow.JobFailed,
		}, err
	}

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
