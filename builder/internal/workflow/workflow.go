package workflow

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

func CreateWorkflowPlan(ctx context.Context, cfg sharedworkflow.WorkflowConfig, source string, workspace string, systemDir string) (*Workflow, error) {
	//TODO: BUILD ID
	buildID := "12345"
	em, err := NewEnvironmentManager(systemDir, buildID)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment manager: %w", err)
	}

	wfCtx := &WorkflowContext{
		Params:             make(map[string]string),
		Workspace:          workspace,
		BuildID:            buildID,
		Source:             source,
		SystemDir:          systemDir,
		EnvironmentManager: em,
		Context:            ctx,
	}

	// set env vars and sets WF_ prefix for workflow envs
	//TODO: better handling of env scopes
	for k, v := range cfg.Env {
		err := em.Set(k, v)
		if err != nil {
			return nil, fmt.Errorf("failed to create workflow, could not set env variable '%s': %w", k, err)
		}
	}
	var jobs []Job

	jobKeys := getJobKeys(cfg.Jobs)

	for _, jobName := range jobKeys {
		j := cfg.Jobs[jobName]
		//TODO: Job env
		// env := make(map[string]string)
		// for k, v := range j.Env {
		// 	env["JOB_"+k] = v
		// }

		var steps []Step
		for _, s := range j.Steps {

			step, err := NewStep(s)
			if err != nil {
				return nil, fmt.Errorf("step '%s' in job '%s' is invalid. %w", s.Name, jobName, err)
			}
			for k, v := range s.Env {
				if err := em.Set(k, v); err != nil {
					return nil, fmt.Errorf("failed to step '%s' while setting env variable '%s': %w ", s.Name, k, err)
				}

			}
			steps = append(steps, step)
		}
		jobs = append(jobs, Job{
			Name:  jobName,
			Needs: []string{},
			Steps: steps,
		})
	}

	wf := &Workflow{
		Name:               cfg.Name,
		Ctx:                wfCtx,
		Jobs:               jobs,
		Results:            make(map[string]map[string]*StepResult),
		EnvironmentManager: em,
	}

	err = wf.setSystemVariables()
	if err != nil {
		return nil, err
	}

	return wf, nil

}

func (w *Workflow) Execute() {
	start := time.Now()
	log.Printf("Starting workflow %s at %s", w.Name, start)

	// Set standard Topi environment variables
	if err := w.setSystemVariables(); err != nil {
		log.Printf("Failed to set system variables: %v", err)
		os.Exit(1)
	}

	for _, job := range w.Jobs {
		log.Printf("Running job %s", job.Name)

		// for _, n := range job.Needs {
		// 	w.Results[job.Name]
		// }
		for _, step := range job.Steps {
			log.Printf("[%s] Running step '%s'", job.Name, step.Name())

			err := w.EnvironmentManager.Load()
			if err != nil {
				log.Printf("Unrecoverable error while loading env vars: %v", err)
				os.Exit(1)
			}
			result, err := step.Exec(w.Ctx)
			if err != nil && job.ErrorPolicy == "Stop" {
				log.Printf("[%s] Step '%s' resulted in an error: %v", job.Name, step.Name(), err)
				os.Exit(1)
			}
			w.setResult(job.Name, step.Name(), result)
			log.Printf("[%s] Finished running '%s', resulted in status: %s", job.Name, step.Name(), result.Status)
		}
	}
	//TODO: better logging, duration and that
}

func getJobKeys(jobs map[string]sharedworkflow.JobConfig) []string {

	j := make([]string, 0, len(jobs))
	for i := range jobs {
		j = append(j, i)
	}

	return j
}

func (wf *Workflow) setResult(jobName, stepName string, result *StepResult) {
	if wf.Results == nil {
		wf.Results = make(map[string]map[string]*StepResult)
	}
	if wf.Results[jobName] == nil {
		wf.Results[jobName] = make(map[string]*StepResult)
	}
	wf.Results[jobName][stepName] = result
}

func (w *Workflow) setSystemVariables() error {

	if err := w.EnvironmentManager.Set("TOPI_WORKSPACE", w.Ctx.Workspace); err != nil {
		return fmt.Errorf("failed to set env TOPI_WORKSPACE: %v", err)
	}
	if err := w.EnvironmentManager.Set("TOPI_BUILDID", w.Ctx.BuildID); err != nil {
		return fmt.Errorf("failed to set env TOPI_BUILDID: %v", err)
	}
	if err := w.EnvironmentManager.Set("TOPI_SYSTEM_DIR", w.Ctx.SystemDir); err != nil {
		return fmt.Errorf("failed to set env TOPI_SYSTEM_DIR: %v", err)
	}
	return nil
}
