package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tobiaskrok/topi/builder/internal/config"
	"github.com/tobiaskrok/topi/builder/internal/utils"
	"github.com/tobiaskrok/topi/builder/internal/workflow"
	_ "github.com/tobiaskrok/topi/builder/internal/workflow/steps/all"
)

// TODO: Environment variables instead of flags
// - Make sure system env variables and user ones do not overlap.

func main() {

	log.Println("Topi Builder starting...")

	repo, rExists := os.LookupEnv("SOURCE_REPO")
	sourceWorkflow, wExists := os.LookupEnv("SOURCE_WORKFLOW")
	// TODO REF

	if !rExists || repo == "" {
		log.Println(fmt.Errorf("system error: env variable SOURCE_REPO was not found"))
		os.Exit(1)
	} else if !utils.ValidateURL(repo) {
		log.Println(fmt.Errorf("system error: malformed repository URL: %s", repo))
		os.Exit(1)
	}

	if !wExists || sourceWorkflow == "" {
		log.Println(fmt.Errorf("system error: env variable SOURCE_WORKFLOW was not found"))
		os.Exit(1)
	}

	// TODO: Implement builder logic
	// - Clone git repository
	// - Load workflow from file or URL based on isWorkflowFile
	// - Execute build based on workflow
	// - Store artifacts to persistent storage
	// - Report build status back

	// Get workspace from environment or use default
	workspace := os.Getenv("TOPI_WORKSPACE")
	if workspace == "" {
		workspace = "/opt/topi/workspace"
	}

	// Get system directory from environment or use default
	systemDir := os.Getenv("TOPI_SYSTEM_DIR")

	if systemDir == "" {
		systemDir = "/opt/topi/system"
	}

	// Get Git token for authentication (optional)
	gitToken := os.Getenv("GIT_TOKEN")

	config, err := config.LoadAndParse(sourceWorkflow)
	if err != nil {
		log.Println(fmt.Errorf("system error: %w", err))
		os.Exit(1)
	}
	wf, err := workflow.CreateWorkflowPlan(config, repo, workspace, systemDir, gitToken)
	if err != nil {
		log.Println(fmt.Errorf("system error: %w", err))
		os.Exit(1)
	}
	log.Println("Created workflow, executing...")
	wf.Execute()
}
