#!/bin/bash
# quick-test.sh - Quick test script for Topi builder development

set -e

# Configuration
REGISTRY_PORT=15001
NAMESPACE=topi-builds
IMAGE_NAME=localhost:${REGISTRY_PORT}/topi-builder:latest

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Function to show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -w, --workflow FILE    Path to workflow file (default: test-workflow.yaml)"
    echo "  -n, --name NAME        Name for the test pod (default: auto-generated)"
    echo "  -b, --build            Build and push image before testing"
    echo "  -f, --follow           Follow logs after creating pod"
    echo "  -c, --clean            Clean up test pods before running"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run with default test-workflow.yaml"
    echo "  $0 -w my-workflow.yaml -b -f         # Build, deploy, and follow logs"
    echo "  $0 -w test-workflows/node.yaml -f    # Test specific workflow"
    echo "  $0 -c -b -f                          # Clean, build, and test"
    exit 0
}

# Parse arguments
WORKFLOW_FILE="test-workflow.yaml"
POD_NAME=""
BUILD=false
FOLLOW=false
CLEAN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -w|--workflow)
            WORKFLOW_FILE="$2"
            shift 2
            ;;
        -n|--name)
            POD_NAME="$2"
            shift 2
            ;;
        -b|--build)
            BUILD=true
            shift
            ;;
        -f|--follow)
            FOLLOW=true
            shift
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Generate pod name if not provided
if [ -z "$POD_NAME" ]; then
    POD_NAME="test-builder-$(date +%s)"
fi

# Clean up if requested
if [ "$CLEAN" = true ]; then
    print_header "Cleaning up test pods"
    kubectl delete pods -l app=topi-builder -n $NAMESPACE --force --grace-period=0 2>/dev/null || true
    print_success "Cleanup complete"
fi

# Build and push if requested
if [ "$BUILD" = true ]; then
    print_header "Building and pushing image"
    
    print_info "Building image..."
    docker build -t $IMAGE_NAME . || {
        print_error "Build failed"
        exit 1
    }
    
    print_info "Pushing to registry..."
    docker push $IMAGE_NAME || {
        print_error "Push failed"
        exit 1
    }
    
    print_success "Image updated"
fi

# Check if workflow file exists
if [ ! -f "$WORKFLOW_FILE" ]; then
    print_error "Workflow file not found: $WORKFLOW_FILE"
    
    # Create a default test workflow if it doesn't exist
    print_info "Creating default test-workflow.yaml..."
    cat > test-workflow.yaml <<'EOF'
version: "1.0"
name: "Test Workflow"

env:
  DEBUG: "true"
  TEST_VAR: "hello"

jobs:
  test:
    steps:
      - name: "Echo Start"
        uses: "echo"
        params:
          message: "Starting test workflow in ${TOPI_WORKSPACE}"
      
      - name: "Run Commands"
        run: |
          echo "Testing shell execution"
          echo "Build ID: ${TOPI_BUILDID}"
          echo "Workspace: ${TOPI_WORKSPACE}"
          echo "System Dir: ${TOPI_SYSTEM_DIR}"
          ls -la ${TOPI_WORKSPACE}/
          
      - name: "Set Environment"
        run: |
          echo "TEST_OUTPUT=success" >> $TOPI_ENV
          echo "TIMESTAMP=$(date +%s)" >> $TOPI_ENV
          
      - name: "Read Environment"
        run: |
          echo "TEST_OUTPUT: $TEST_OUTPUT"
          echo "TIMESTAMP: $TIMESTAMP"
      
      - name: "Echo Complete"
        uses: "echo"
        params:
          message: "Workflow completed at $(date)!"
EOF
    print_success "Created default test-workflow.yaml"
    WORKFLOW_FILE="test-workflow.yaml"
fi

# Create/update ConfigMap with the workflow
print_header "Uploading workflow to ConfigMap"

CONFIGMAP_NAME="workflow-${POD_NAME}"

kubectl create configmap $CONFIGMAP_NAME \
    --from-file=workflow.yaml=$WORKFLOW_FILE \
    -n $NAMESPACE \
    --dry-run=client -o yaml | kubectl apply -f -

print_success "ConfigMap created: $CONFIGMAP_NAME"

# Create the test pod
print_header "Creating test pod: $POD_NAME"

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: $POD_NAME
  namespace: $NAMESPACE
  labels:
    app: topi-builder
    test: "true"
    workflow: "$(basename $WORKFLOW_FILE .yaml)"
spec:
  containers:
  - name: builder
    image: $IMAGE_NAME
    imagePullPolicy: Always
    env:
    - name: SOURCE_REPO
      value: "http://localhost/topi/topi-test"
    - name: SOURCE_WORKFLOW
      value: "/workflow/workflow.yaml"
    - name: TOPI_BUILDID
      value: "$POD_NAME"
    - name: TOPI_WORKSPACE
      value: "/opt/topi/workspace"
    - name: TOPI_SYSTEM_DIR
      value: "/opt/topi/system"
    - name: MINIO_ENDPOINT
      value: "minio.topi-system:9000"
    - name: MINIO_ACCESS_KEY
      value: "minioadmin"
    - name: MINIO_SECRET_KEY
      value: "minioadmin123"
    - name: GIT_TOKEN
      value: "098d57492d74f5ae007e0069de6cf868d166f9a8"
    volumeMounts:
    - name: workspace
      mountPath: /opt/topi/workspace
    - name: system
      mountPath: /opt/topi/system
    - name: workflow
      mountPath: /workflow
  volumes:
  - name: workspace
    emptyDir: {}
  - name: system
    emptyDir: {}
  - name: workflow
    configMap:
      name: $CONFIGMAP_NAME
  restartPolicy: Never
EOF

print_success "Pod created: $POD_NAME"

# Follow logs if requested
if [ "$FOLLOW" = true ]; then
    print_header "Following logs"
    print_info "Waiting for pod to start..."
    
    # Wait for pod to be ready
    kubectl wait --for=condition=Ready pod/$POD_NAME -n $NAMESPACE --timeout=30s 2>/dev/null || true
    
    # Follow logs
    kubectl logs -f $POD_NAME -n $NAMESPACE
    
    # Show pod status after completion
    echo ""
    print_header "Pod Status"
    kubectl get pod $POD_NAME -n $NAMESPACE
else
    echo ""
    print_info "Pod created. To view logs, run:"
    echo "  kubectl logs -f $POD_NAME -n $NAMESPACE"
    echo ""
    print_info "To check status:"
    echo "  kubectl get pod $POD_NAME -n $NAMESPACE"
fi

# Cleanup instructions
echo ""
print_info "To clean up this test pod:"
echo "  kubectl delete pod $POD_NAME -n $NAMESPACE"
echo ""
print_info "To clean up all test pods:"
echo "  kubectl delete pods -l app=topi-builder -n $NAMESPACE"
