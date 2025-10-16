#!/bin/bash
# run-dev.sh - Complete development environment setup for Topi

set -e

# Configuration
REGISTRY_PORT=15001
CLUSTER_NAME=topi-dev
REGISTRY_NAME=kind-registry

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Cleanup function
cleanup() {
    print_header "Cleaning up existing resources"
    
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        print_info "Deleting existing Kind cluster..."
        kind delete cluster --name $CLUSTER_NAME
    fi
    
    if docker ps -a | grep -q $REGISTRY_NAME; then
        print_info "Removing existing registry container..."
        docker rm -f $REGISTRY_NAME
    fi
    
    print_success "Cleanup complete"
}

# Main setup
main() {
    print_header "Topi Development Environment Setup"
    echo "Registry Port: $REGISTRY_PORT"
    echo "Cluster Name: $CLUSTER_NAME"
    echo ""

    # Check for required tools
    print_info "Checking required tools..."
    for tool in docker kind kubectl; do
        if ! command -v $tool &> /dev/null; then
            print_error "$tool is not installed. Please install it first."
            exit 1
        fi
    done
    print_success "All required tools found"

    # Clean up if requested
    if [ "$1" == "--clean" ]; then
        cleanup
    fi

    # Create Kind cluster
    print_header "Creating Kind cluster"
    cat <<EOF | kind create cluster --name $CLUSTER_NAME --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  # MinIO ports (optional, for direct access)
  - containerPort: 30900
    hostPort: 9000
    protocol: TCP
  - containerPort: 30901
    hostPort: 9001
    protocol: TCP
- role: worker
  extraMounts:
  # Mount for build cache
  - hostPath: /tmp/topi-cache
    containerPath: /cache
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REGISTRY_PORT}"]
    endpoint = ["http://${REGISTRY_NAME}:5000"]
EOF
    print_success "Kind cluster created"

    # Start Docker registry
    print_header "Starting Docker Registry"
    
    # Remove any existing registry container first
    docker rm -f $REGISTRY_NAME 2>/dev/null || true
    
    docker run -d \
        --restart=always \
        --name $REGISTRY_NAME \
        -p ${REGISTRY_PORT}:5000 \
        registry:2
    
    # Connect registry to Kind network
    docker network connect kind $REGISTRY_NAME || true
    
    # Document the registry
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
    print_success "Docker registry started on port $REGISTRY_PORT"

    # Create namespaces
    print_header "Creating namespaces"
    kubectl create namespace topi-system --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace topi-builds --dry-run=client -o yaml | kubectl apply -f -
    print_success "Namespaces created"

    # Deploy MinIO
    print_header "Deploying MinIO for artifact storage"
    kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: minio-pvc
  namespace: topi-system
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
  namespace: topi-system
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
          value: "minioadmin"
        - name: MINIO_ROOT_PASSWORD
          value: "minioadmin123"
        - name: MINIO_DEFAULT_BUCKETS
          value: "artifacts,cache,logs"
        ports:
        - containerPort: 9000
          name: api
        - containerPort: 9001
          name: console
        volumeMounts:
        - name: storage
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /minio/health/live
            port: 9000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /minio/health/ready
            port: 9000
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: minio-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: minio
  namespace: topi-system
spec:
  type: NodePort
  ports:
  - port: 9000
    targetPort: 9000
    nodePort: 30900
    name: api
  - port: 9001
    targetPort: 9001
    nodePort: 30901
    name: console
  selector:
    app: minio
---
apiVersion: v1
kind: Secret
metadata:
  name: minio-credentials
  namespace: topi-system
type: Opaque
stringData:
  access-key: minioadmin
  secret-key: minioadmin123
  endpoint: minio:9000
EOF
    
    print_info "Waiting for MinIO to be ready..."
    kubectl wait --for=condition=available --timeout=120s deployment/minio -n topi-system
    print_success "MinIO deployed successfully"

    # Build and push builder image
    print_header "Building and pushing Builder image"

    if [ ! -f "./builder/Dockerfile" ]; then
        print_error "Builder Dockerfile not found. Are you in the project root?"
        exit 1
    fi

    print_info "Building builder image..."
    docker build -t localhost:${REGISTRY_PORT}/topi-builder:latest -f builder/Dockerfile .

    print_info "Pushing to local registry..."
    docker push localhost:${REGISTRY_PORT}/topi-builder:latest
    print_success "Builder image ready"

    # Create RBAC for builder
    print_header "Setting up RBAC"
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: topi-builder
  namespace: topi-builds
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: topi-builder
  namespace: topi-builds
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["secrets", "configmaps"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: topi-builder
  namespace: topi-builds
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: topi-builder
subjects:
- kind: ServiceAccount
  name: topi-builder
  namespace: topi-builds
EOF
    print_success "RBAC configured"

    # Create test BuildJob pod
    print_header "Creating test Builder pod"
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-builder-$(date +%s)
  namespace: topi-builds
  labels:
    app: topi-builder
    test: "true"
spec:
  serviceAccountName: topi-builder
  containers:
  - name: builder
    image: localhost:${REGISTRY_PORT}/topi-builder:latest
    imagePullPolicy: Always
    env:
    - name: SOURCE_REPO
      value: "https://github.com/example/test"
    - name: SOURCE_WORKFLOW
      value: "./test-workflow.yaml"
    - name: BUILD_ID
      value: "test-$(date +%s)"
    - name: WORKSPACE
      value: "/opt/topi-dev"
    - name: MINIO_ENDPOINT
      value: "minio.topi-system:9000"
    - name: MINIO_ACCESS_KEY
      value: "minioadmin"
    - name: MINIO_SECRET_KEY
      value: "minioadmin123"
    - name: REGISTRY_ENDPOINT
      value: "localhost:${REGISTRY_PORT}"
    volumeMounts:
    - name: workspace
      mountPath: /opt/topi-dev
    - name: cache
      mountPath: /cache
  volumes:
  - name: workspace
    emptyDir: {}
  - name: cache
    emptyDir: {}
  restartPolicy: Never
EOF
    print_success "Test builder pod created"

    # Print summary
    print_header "Setup Complete!"
    echo ""
    echo -e "${GREEN}🎉 Topi development environment is ready!${NC}"
    echo ""
    echo "📦 Registry: localhost:${REGISTRY_PORT}"
    echo "🗄️  MinIO API: http://localhost:9000 (minioadmin/minioadmin123)"
    echo "🖥️  MinIO Console: http://localhost:9001 (minioadmin/minioadmin123)"
    echo ""
    echo -e "${YELLOW}Useful commands:${NC}"
    echo "  Watch logs:        kubectl logs -f -l app=topi-builder -n topi-builds"
    echo "  List pods:         kubectl get pods -n topi-builds"
    echo "  MinIO console:     kubectl port-forward -n topi-system svc/minio 9001:9001"
    echo "  Delete test pod:   kubectl delete pod -l test=true -n topi-builds"
    echo "  Rebuild & test:    ./run-dev.sh --rebuild"
    echo "  Clean everything:  ./run-dev.sh --clean"
    echo ""
    echo -e "${YELLOW}Quick test:${NC}"
    echo "  kubectl logs -f \$(kubectl get pod -l app=topi-builder -n topi-builds -o name | head -1) -n topi-builds"
}

# Handle rebuild flag
if [ "$1" == "--rebuild" ]; then
    print_header "Rebuilding and redeploying"
    
    # Delete existing test pods
    kubectl delete pod -l test=true -n topi-builds 2>/dev/null || true

    # Rebuild and push
    docker build -t localhost:${REGISTRY_PORT}/topi-builder:latest -f builder/Dockerfile .
    docker push localhost:${REGISTRY_PORT}/topi-builder:latest
    
    # Create new test pod
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-builder-$(date +%s)
  namespace: topi-builds
  labels:
    app: topi-builder
    test: "true"
spec:
  containers:
  - name: builder
    image: localhost:${REGISTRY_PORT}/topi-builder:latest
    imagePullPolicy: Always
    env:
    - name: SOURCE_REPO
      value: "https://github.com/example/test"
    - name: SOURCE_WORKFLOW
      value: "./test-workflow.yaml"
    - name: BUILD_ID
      value: "test-$(date +%s)"
    - name: WORKSPACE
      value: "/opt/topi-dev"
    volumeMounts:
    - name: workspace
      mountPath: /opt/topi-dev
  volumes:
  - name: workspace
    emptyDir: {}
  restartPolicy: Never
EOF
    
    print_success "Rebuild complete"
    echo "Watch logs: kubectl logs -f \$(kubectl get pod -l app=topi-builder -n topi-builds -o name | head -1) -n topi-builds"
    exit 0
fi

# Handle clean flag
if [ "$1" == "--clean" ]; then
    cleanup
    exit 0
fi

# Run main setup
main