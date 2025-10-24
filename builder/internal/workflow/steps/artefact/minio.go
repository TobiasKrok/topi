package artefact

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/tobiaskrok/topi/builder/internal/utils"
	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

func init() {
	workflow.RegisterStep("artefact/minio", newMinioArtefactStep)
}

type MinioArtefactStep struct {
	name   string
	source string

	// added during execution
	secretKey string
	accessKey string
	bucket    string
	endpoint  string
	owner     string
	repo      string
}

func newMinioArtefactStep(cfg sharedworkflow.StepConfig) (workflow.Step, error) {
	// its okay if these are empty as there are defaults

	return &MinioArtefactStep{
		name:   cfg.Name,
		source: cfg.Params["source"],
	}, nil
}

func (s *MinioArtefactStep) Exec(ctx *workflow.WorkflowContext) (*workflow.StepResult, error) {

	err := s.setConfiguration(ctx)
	if err != nil {
		log.Printf("[minio] Configuration error: %v", err)
		return &workflow.StepResult{
			Status: workflow.JobFailed,
		}, err
	}
	client, err := minio.New(s.endpoint, &minio.Options{Creds: credentials.NewStaticV4(s.accessKey, s.secretKey, "")})

	if err != nil {
		log.Printf("[minio] Failed to initialize MinIO client: %v", err)
		return &workflow.StepResult{
			Status: workflow.JobFailed,
		}, err
	}
	exists, err := client.BucketExists(ctx.Context, s.bucket)
	if err != nil {
		log.Printf("[minio] Failed to check if bucket exists: %v", err)
		return &workflow.StepResult{
			Status: workflow.JobFailed,
		}, err
	}

	if !exists {
		log.Printf("[minio] Bucket %s does not exist", s.bucket)
		return &workflow.StepResult{
			Status: workflow.JobFailed,
		}, err
	}

	err = s.zipAndUpload(client, ctx.Context)

	if err != nil {
		log.Printf("[minio] Failed to upload artefact: %v", err)
		return &workflow.StepResult{
			Status: workflow.JobFailed,
		}, err
	}

	log.Printf("[minio] Successfully uploaded artefact to %s/%s/%s", s.bucket, s.owner, s.repo)
	return &workflow.StepResult{
		Status: workflow.JobSuccess,
	}, nil
}

func (s *MinioArtefactStep) Name() string {
	if s.name == "" {

		return s.name
	}
	return "Upload MinIO Artefact"
}

func (s *MinioArtefactStep) Description() string {
	return "Uploads artefacts to MinIO"
}

func (s *MinioArtefactStep) setConfiguration(ctx *workflow.WorkflowContext) error {

	key, err := ctx.EnvironmentManager.Get("MINIO_KEY")
	if err != nil {
		return fmt.Errorf("missing environment variable MINIO_KEY")
	}
	id, err := ctx.EnvironmentManager.Get("MINIO_ID")
	if err != nil {
		return fmt.Errorf("missing environment variable MINIO_ID")
	}

	endpoint, err := ctx.EnvironmentManager.Get("MINIO_ENDPOINT")
	if err != nil {
		return fmt.Errorf("missing environment variable MINIO_ENDPOINT")
	}

	bucket, err := ctx.EnvironmentManager.Get("MINIO_ARTEFACT_BUCKET")
	if err != nil {
		return fmt.Errorf("missing environment variable MINIO_BUCKET")
	}

	owner, repo, err := utils.ParseGitURL(ctx.Source)
	if err != nil {
		return fmt.Errorf("failed to parse source URL '%s': %w", ctx.Source, err)
	}

	if s.source == "" {
		s.source = ctx.Workspace
	}

	s.accessKey = id
	s.secretKey = key
	s.endpoint = endpoint
	s.bucket = bucket
	s.owner = owner
	s.repo = repo
	return nil
}

func (s *MinioArtefactStep) zipAndUpload(client *minio.Client, ctx context.Context) error {
	// Create temp tar.gz file
	tarFile, err := os.CreateTemp("", "upload-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk the folder and add files to tar
	err = filepath.Walk(s.source, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(s.source, file)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	tarWriter.Close()
	gzWriter.Close()

	objectPath := fmt.Sprintf("/%s/%s/artefact.tar.gz", s.owner, s.repo)
	_, err = client.FPutObject(ctx, s.bucket, objectPath, tarFile.Name(),
		minio.PutObjectOptions{
			ContentType: "application/gzip",
		})

	return err
}
