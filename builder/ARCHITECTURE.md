# Topi Builder - Architecture & Implementation Guide

## System Overview

Topi is a Kubernetes-native CI/CD system with three core components:

```
Git Repository → Engine → Scheduler → Builder → Artifacts → Deployment
                   ↓         ↓           ↓
              (Monitors) (Creates)  (Executes)
                        BuildJob CRD
```

### Components
1. **Engine**: Monitors Git repositories for changes (`.topi` folder detection)
2. **Scheduler**: Creates and manages BuildJob CRDs and Kubernetes Jobs
3. **Builder**: Executes workflows defined in YAML within isolated containers

## Artifact Storage Architecture

### Option 1: S3-Compatible Storage (MinIO) - RECOMMENDED

**Why this approach:**
- Industry-standard S3 API (transferable skills)
- Works locally with Kind/Minikube
- Easy migration to cloud providers (AWS S3, GCS, Azure Blob)

**Implementation:**
```yaml
# deploy/minio.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
spec:
  replicas: 1
  selector:
    matchLabels:
      app: minio
  template:
    metadata:
      labels:
        app: minio
    spec:
      containers:
      - name: minio
        image: minio/minio:latest
        args:
        - server
        - /data
        - --console-address
        - ":9001"
        env:
        - name: MINIO_ROOT_USER
          value: "admin"
        - name: MINIO_ROOT_PASSWORD
          value: "admin123"
        ports:
        - containerPort: 9000
        - containerPort: 9001
        volumeMounts:
        - name: storage
          mountPath: /data
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: minio-pvc
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: minio-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

**Artifact Structure:**
```
bucket/
├── builds/
│   ├── {build-id}/
│   │   ├── artifacts/
│   │   │   ├── app.tar.gz
│   │   │   └── manifest.json
│   │   └── logs/
│   │       └── build.log
```

### Option 2: Kubernetes PersistentVolume with NFS

**Why this approach:**
- Simple, no additional services needed
- Good for learning Kubernetes storage concepts
- Direct file access for debugging

**Implementation:**
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: artifacts-storage
spec:
  accessModes:
    - ReadWriteMany  # Important: allows multiple pods to write
  resources:
    requests:
      storage: 20Gi
```

**Mount in Builder Pod:**
```yaml
volumeMounts:
- name: artifacts
  mountPath: /artifacts
volumes:
- name: artifacts
  persistentVolumeClaim:
    claimName: artifacts-storage
```

### Option 3: OCI Registry for Everything

**Why this approach:**
- Modern: Everything is a container/OCI artifact
- Single storage solution for images and artifacts
- Built-in versioning and distribution

**Implementation:**
Use ORAS (OCI Registry As Storage) to push any artifact:
```bash
# Push artifacts as OCI artifacts
oras push localhost:5000/artifacts/myapp:v1.0.0 \
  --artifact-type application/vnd.myapp \
  ./build/app.tar.gz:application/gzip
```

## Docker-in-Docker (DinD) Considerations

### The Challenge
Your builder runs in a container. Building Docker images inside a container requires special setup.

### Solution 1: Docker-in-Docker (DinD) - NOT RECOMMENDED

**How it works:**
```yaml
# Builder pod with DinD
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: builder
    image: docker:dind
    securityContext:
      privileged: true  # SECURITY RISK!
    env:
    - name: DOCKER_TLS_CERTDIR
      value: ""
```

**Problems:**
- Requires privileged mode (security risk)
- Complex networking
- Cache issues
- Storage driver conflicts

### Solution 2: Kaniko - RECOMMENDED for Kubernetes

**How it works:**
Kaniko builds container images without Docker daemon:

```yaml
# Workflow step for Kaniko
- name: "Build Container"
  uses: "kaniko"
  params:
    context: "."
    dockerfile: "./Dockerfile"
    destination: "registry.local:5000/myapp:latest"
```

**Implementation in Builder:**
```go
// internal/workflow/steps/container/kaniko.go
type KanikoStep struct {
    Context     string   `yaml:"context"`
    Dockerfile  string   `yaml:"dockerfile"`
    Destination []string `yaml:"destination"`
    Cache       bool     `yaml:"cache"`
}

func (k *KanikoStep) Exec(ctx *WorkflowContext) (*StepResult, error) {
    cmd := exec.Command("/kaniko/executor",
        "--context", k.Context,
        "--dockerfile", k.Dockerfile,
        "--destination", k.Destination[0],
    )
    // ... execute and handle results
}
```

### Solution 3: BuildKit - Modern Alternative

**How it works:**
BuildKit can run as a service in your cluster:

```yaml
# BuildKit as a service
apiVersion: apps/v1
kind: Deployment
metadata:
  name: buildkitd
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: buildkitd
        image: moby/buildkit:master
        args:
        - --addr
        - tcp://0.0.0.0:1234
        securityContext:
          privileged: true
```

**Use from Builder:**
```bash
buildctl --addr tcp://buildkitd:1234 \
  build \
  --frontend dockerfile.v0 \
  --local context=. \
  --local dockerfile=. \
  --output type=image,name=registry:5000/myapp:latest,push=true
```

### Solution 4: Mount Docker Socket - For Development Only

**How it works:**
```yaml
# docker-compose.test.yml modification
services:
  builder-test:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - .:/builder
```

**In workflow:**
```yaml
- name: "Build with host Docker"
  run: |
    docker build -t myapp:latest .
    docker push localhost:5000/myapp:latest
```

**Note:** Only works in development, not in Kubernetes pods.

## Deployment Foundation

### Phase 1: Direct Kubernetes Deployment

**Workflow Steps:**
```yaml
jobs:
  deploy:
    steps:
      - name: "Deploy to Kubernetes"
        uses: "kubectl"
        params:
          action: "apply"
          manifest: "./k8s/deployment.yaml"
          namespace: "staging"
```

**Implementation:**
```go
type KubectlStep struct {
    Action    string `yaml:"action"`    // apply, delete, rollout
    Manifest  string `yaml:"manifest"`
    Namespace string `yaml:"namespace"`
}
```

### Phase 2: GitOps Integration

**Architecture:**
```
Builder → Push Artifacts → Update Manifest Repo → ArgoCD/Flux → Deploy
```

**Benefits:**
- Git as source of truth
- Automatic rollback
- Audit trail
- No direct cluster access needed

### Phase 3: Progressive Deployment

**Strategies to implement:**
1. **Blue-Green**: Two identical environments, switch traffic
2. **Canary**: Gradual rollout to percentage of users
3. **Rolling**: Replace instances one by one

## Implementation Roadmap

### Week 1-2: Basic Artifact Storage
- [ ] Deploy MinIO to Kind cluster
- [ ] Add `artifact-upload` step type
- [ ] Store artifact URLs in BuildJob status
- [ ] Test with simple file uploads

### Week 3-4: Container Building
- [ ] Implement Kaniko step
- [ ] Set up local registry (localhost:5001)
- [ ] Build and push test images
- [ ] Add image scanning (Trivy)

### Week 5-6: Basic Deployment
- [ ] Create `kubectl-apply` step
- [ ] Add namespace management
- [ ] Implement basic health checks
- [ ] Add rollback capability

### Week 7-8: Advanced Features
- [ ] Multi-environment support
- [ ] GitOps with ArgoCD
- [ ] Deployment strategies
- [ ] Monitoring integration

## Example Workflows

### Build and Store Artifacts
```yaml
version: "1.0"
name: "Build and Store Application"

env:
  REGISTRY: "localhost:5001"
  BUCKET: "artifacts"

jobs:
  build:
    steps:
      - name: "Compile Application"
        run: |
          go build -o app cmd/main.go
          tar czf app.tar.gz app

      - name: "Upload to MinIO"
        uses: "minio-upload"
        params:
          source: "./app.tar.gz"
          bucket: "${BUCKET}"
          path: "builds/${TOPI_BUILDID}/"

      - name: "Build Container"
        uses: "kaniko"
        params:
          context: "."
          dockerfile: "./Dockerfile"
          destination: "${REGISTRY}/myapp:${TOPI_BUILDID}"

      - name: "Update Manifest"
        run: |
          sed -i "s|image:.*|image: ${REGISTRY}/myapp:${TOPI_BUILDID}|" k8s/deployment.yaml
          
      - name: "Store Manifest"
        uses: "minio-upload"
        params:
          source: "./k8s/deployment.yaml"
          bucket: "${BUCKET}"
          path: "manifests/${TOPI_BUILDID}/"
```

### Deploy to Kubernetes
```yaml
version: "1.0"
name: "Deploy Application"

jobs:
  deploy:
    steps:
      - name: "Download Manifest"
        uses: "minio-download"
        params:
          bucket: "artifacts"
          path: "manifests/${DEPLOY_VERSION}/deployment.yaml"
          destination: "./deployment.yaml"

      - name: "Apply to Staging"
        uses: "kubectl"
        params:
          action: "apply"
          manifest: "./deployment.yaml"
          namespace: "staging"

      - name: "Wait for Rollout"
        uses: "kubectl"
        params:
          action: "rollout"
          resource: "deployment/myapp"
          namespace: "staging"
          wait: true
          timeout: "5m"

      - name: "Health Check"
        run: |
          kubectl -n staging wait --for=condition=ready pod -l app=myapp --timeout=300s
```

## BuildJob CRD Enhancement

```go
type BuildJobSpec struct {
    Repository   string `json:"repository"`
    Branch       string `json:"branch"`
    WorkflowPath string `json:"workflowPath"`
    
    // New fields for artifacts and deployment
    Artifacts ArtifactSpec   `json:"artifacts,omitempty"`
    Deploy    DeploymentSpec `json:"deploy,omitempty"`
}

type ArtifactSpec struct {
    Storage  string   `json:"storage"`  // "minio", "nfs", "registry"
    Bucket   string   `json:"bucket"`
    Preserve int      `json:"preserve"` // Keep last N builds
}

type DeploymentSpec struct {
    Environments []EnvironmentSpec `json:"environments"`
    Strategy     string            `json:"strategy"` // "direct", "gitops"
    AutoDeploy   []string          `json:"autoDeploy"` // ["dev", "staging"]
}

type BuildJobStatus struct {
    Phase      string         `json:"phase"`
    StartTime  *metav1.Time   `json:"startTime,omitempty"`
    EndTime    *metav1.Time   `json:"endTime,omitempty"`
    
    // New status fields
    Artifacts  []ArtifactInfo `json:"artifacts,omitempty"`
    Images     []ImageInfo    `json:"images,omitempty"`
    Deployment DeploymentInfo `json:"deployment,omitempty"`
}

type ArtifactInfo struct {
    Name     string `json:"name"`
    Type     string `json:"type"`
    Location string `json:"location"`
    Size     int64  `json:"size"`
    Checksum string `json:"checksum"`
}
```

## Security Considerations

### Secrets Management
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: builder-credentials
type: Opaque
data:
  minio-access-key: <base64>
  minio-secret-key: <base64>
  registry-password: <base64>
```

### RBAC for Builder
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: builder-role
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["create", "update", "patch"]
```

## Testing in Development

### Local Testing with Docker
```bash
# Test with MinIO in Docker
docker run -d \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=admin \
  -e MINIO_ROOT_PASSWORD=admin123 \
  minio/minio server /data --console-address ":9001"

# Test Kaniko build
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context=/workspace \
  --dockerfile=/workspace/Dockerfile \
  --no-push
```

### Kind Cluster Setup
```bash
# Create cluster with registry
kind create cluster --config=- <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:5001"]
    endpoint = ["http://kind-registry:5000"]
nodes:
- role: control-plane
- role: worker
  extraMounts:
  - hostPath: /tmp/artifacts
    containerPath: /artifacts
EOF

# Run local registry
docker run -d --restart=always -p 5001:5000 --name kind-registry registry:2
```

## Next Steps

1. **Start Simple**: Get basic artifact upload working with MinIO
2. **Add Container Builds**: Implement Kaniko for in-cluster builds
3. **Basic Deployments**: kubectl apply from workflows
4. **Iterate**: Add GitOps, monitoring, advanced deployments

## Resources

- [Kaniko Documentation](https://github.com/GoogleContainerTools/kaniko)
- [BuildKit Documentation](https://github.com/moby/buildkit)
- [MinIO Kubernetes Guide](https://min.io/docs/minio/kubernetes/upstream/)
- [ArgoCD Getting Started](https://argo-cd.readthedocs.io/en/stable/getting_started/)
- [ORAS (OCI Registry As Storage)](https://oras.land/)

## FAQ

### Q: Can I run Docker inside my Builder container?
**A:** Yes, but it's not recommended. Use Kaniko or BuildKit instead. They're designed for containerized environments and don't require privileged access.

### Q: Should I use PVC or S3 for artifacts?
**A:** Start with MinIO (S3-compatible) as it teaches industry-standard patterns and easily migrates to cloud providers.

### Q: How do I handle large artifacts?
**A:** 
- Use streaming uploads for S3/MinIO
- Implement chunked uploads
- Consider artifact expiration policies
- Use CDN for distribution if needed

### Q: How do I test locally before deploying to Kubernetes?
**A:** Use the Docker test environment with mounted volumes to simulate artifact storage:
```bash
make docker-test WORKFLOW=test-workflows/build-and-store.yaml
```