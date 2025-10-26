#!/bin/sh
set -o errexit

# =============================================================================
# Topi Complete Development Environment Setup
# =============================================================================
# This script sets up the complete Topi development environment including:
# - Kind cluster with local registry
# - MinIO Operator and Tenant
# - Required namespaces and secrets
# - CRD installations
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Configuration
CLUSTER_NAME='topi-dev'
MINIO_NAMESPACE='topi-system'
BUILDS_NAMESPACE='topi-builds'

# MinIO Configuration (modify as needed)
MINIO_STORAGE_SIZE='5Gi'
MINIO_ROOT_USER='minioadmin'
MINIO_ROOT_PASSWORD='minioadmin123'

# Gitea Token (modify with your actual token)
# Generate a token in Gitea: Settings -> Applications -> Generate New Token
GIT_TOKEN='59421341647617eecb93b07b731d4c1a1b6a1c7c'

echo "🚀 Starting Topi Development Environment Setup"
echo "=============================================="
echo ""

# =============================================================================
# 1. Create Kind Cluster with Registry
# =============================================================================
echo "📦 Step 1/5: Setting up Kind cluster..."
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  echo "   ⚠️  Cluster '${CLUSTER_NAME}' already exists!"
  read -p "   Do you want to delete and recreate it? (y/N): " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "   🗑️  Deleting existing cluster..."
    kind delete cluster --name ${CLUSTER_NAME}
  else
    echo "   ℹ️  Using existing cluster"
    kubectl cluster-info --context kind-${CLUSTER_NAME}
  fi
fi

if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  "${SCRIPT_DIR}/kind.sh"
fi
echo ""

# =============================================================================
# 2. Create Namespaces
# =============================================================================
echo "📁 Step 2/5: Creating namespaces..."
kubectl create namespace ${MINIO_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace ${BUILDS_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
echo "   ✅ Namespaces created"
echo ""

# =============================================================================
# 3. Install MinIO Operator using Helm
# =============================================================================
echo "🗄️  Step 3/5: Installing MinIO Operator with Helm..."
if helm list -n minio-operator | grep -q minio-operator; then
  echo "   ℹ️  MinIO Operator already installed"
else
  helm repo add minio-operator https://operator.min.io/ 2>/dev/null || true
  helm repo update
  helm install \
    --namespace minio-operator \
    --create-namespace \
    minio-operator minio-operator/operator

  echo "   ⏳ Waiting for operator to be ready..."
  kubectl wait --for=condition=available --timeout=300s deployment/minio-operator -n minio-operator
  echo "   ✅ MinIO Operator installed"
fi
echo ""

# =============================================================================
# 4. Create MinIO Tenant using Helm
# =============================================================================
echo "🗄️  Step 4/5: Creating MinIO Tenant with Helm..."
if helm list -n ${MINIO_NAMESPACE} | grep -q minio-tenant; then
  echo "   ℹ️  MinIO Tenant already installed"
else
  helm repo add minio-tenant https://operator.min.io/ 2>/dev/null || true
  helm repo update

  helm install \
    --namespace ${MINIO_NAMESPACE} \
    --create-namespace \
    minio-tenant minio-operator/tenant \
    --set tenant.name=minio-tenant \
    --set tenant.pools[0].name=pool-0 \
    --set tenant.pools[0].servers=1 \
    --set tenant.pools[0].volumesPerServer=1 \
    --set tenant.pools[0].size=${MINIO_STORAGE_SIZE} \
    --set tenant.configSecret.name=minio-env-configuration \
    --set tenant.configSecret.accessKey=${MINIO_ROOT_USER} \
    --set tenant.configSecret.secretKey=${MINIO_ROOT_PASSWORD} \
    --set tenant.mountPath=/export \
    --set tenant.requestAutoCert=false \
    --set ingress.api.enabled=false \
    --set ingress.console.enabled=false

  echo "   ⏳ Waiting for MinIO Tenant StatefulSet to be created..."
  # Wait for the StatefulSet to exist first
  for i in {1..60}; do
    if kubectl get statefulset minio-tenant-pool-0 -n ${MINIO_NAMESPACE} &>/dev/null; then
      echo "   ✅ StatefulSet created"
      break
    fi
    if [ $i -eq 60 ]; then
      echo "   ⚠️  Timeout waiting for StatefulSet to be created"
      kubectl get all -n ${MINIO_NAMESPACE}
      exit 1
    fi
    sleep 2
  done

  # Now wait for StatefulSet to be ready
  echo "   ⏳ Waiting for MinIO StatefulSet to be ready (this may take a few minutes)..."
  kubectl wait --for=jsonpath='{.status.readyReplicas}'=1 \
    statefulset/minio-tenant-pool-0 \
    -n ${MINIO_NAMESPACE} \
    --timeout=300s || {
    echo "   ⚠️  Timeout waiting for StatefulSet to be ready, checking status..."
    kubectl get pods -n ${MINIO_NAMESPACE}
    kubectl describe statefulset minio-tenant-pool-0 -n ${MINIO_NAMESPACE}
    echo "   ⚠️  MinIO may take longer to initialize. Check the logs with:"
    echo "       kubectl logs -n ${MINIO_NAMESPACE} minio-tenant-pool-0-0"
  }

  echo "   ✅ MinIO Tenant created"
fi

# Create NodePort services for MinIO access from host
echo "   🔌 Creating NodePort services for external access..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: minio-external
  namespace: ${MINIO_NAMESPACE}
spec:
  type: NodePort
  ports:
  - name: http
    port: 9000
    targetPort: 9000
    nodePort: 30900
  selector:
    v1.min.io/tenant: minio-tenant
    v1.min.io/pool: pool-0
---
apiVersion: v1
kind: Service
metadata:
  name: minio-console-external
  namespace: ${MINIO_NAMESPACE}
spec:
  type: NodePort
  ports:
  - name: http
    port: 9090
    targetPort: 9090
    nodePort: 30901
  selector:
    v1.min.io/tenant: minio-tenant
    v1.min.io/pool: pool-0
EOF
echo "   ✅ External services created"
echo ""

# =============================================================================
# 5. Create Additional Secrets
# =============================================================================
echo "🔑 Step 5/5: Creating additional secrets..."

# MinIO credentials secret and config for Topi to use
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: provider-minio
  namespace: ${BUILDS_NAMESPACE}
type: Opaque
stringData:
  accessKey: "q1OBTGdrtZ2VYPS9ZmEP"
  accessSecret: "azCNpVNse84J9LmnVi0jAz2e15acHyGoZZO10fTb"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: provider-minio
  namespace: ${BUILDS_NAMESPACE}
data:
  endpoint: "minio-tenant-hl.${MINIO_NAMESPACE}.svc.cluster.local:9000"
  bucket: "artefact"
  use-ssl: "false"
EOF

# Git provider token
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: provider-git-token
  namespace: ${BUILDS_NAMESPACE}
type: Opaque
stringData:
  token: "${GIT_TOKEN}"
EOF
echo "   ✅ Secrets created"
echo ""

# =============================================================================
# Setup Complete!
# =============================================================================
echo ""
echo "✨ ============================================== ✨"
echo "   Topi Development Environment Ready!"
echo "✨ ============================================== ✨"
echo ""
echo "📋 Environment Information:"
echo "   Cluster:      ${CLUSTER_NAME}"
echo "   MinIO Console: http://localhost:9001 (HTTP - no TLS)"
echo "   MinIO API (internal): http://minio-tenant-hl.${MINIO_NAMESPACE}.svc.cluster.local:9000"
echo "   Credentials:  ${MINIO_ROOT_USER} / ${MINIO_ROOT_PASSWORD}"
echo ""
echo "🔧 Verify Installation:"
echo "   kubectl get nodes"
echo "   kubectl get pods -n ${MINIO_NAMESPACE}"
echo "   kubectl get pods -n ${BUILDS_NAMESPACE}"
echo ""
echo "📝 Access MinIO Console:"
echo "   URL:      http://localhost:9001"
echo "   Username: ${MINIO_ROOT_USER}"
echo "   Password: ${MINIO_ROOT_PASSWORD}"
echo "   Note:     Running without TLS for development simplicity"
echo ""
echo "🎯 Next Steps:"
echo "   1. Install Topi CRDs:"
echo "      make install-crds"
echo ""
echo "   2. Build and push images:"
echo "      make kind-push-all"
echo ""
echo "   3. Deploy Topi Operator:"
echo "      make kind-deploy-scheduler"
echo ""
echo "   4. Connect Gitea to Kind network (if not already connected):"
echo "      docker network connect kind gitea"
echo ""
