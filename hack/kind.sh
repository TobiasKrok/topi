
#!/bin/sh
set -o errexit

# =============================================================================
# Kind Cluster Setup Script for Topi Development
# =============================================================================
# This script sets up a local Kind cluster with:
# - Local Docker registry for fast image development
# - Multi-node cluster (1 control-plane + workers)
# - Port mappings for MinIO and other services
# =============================================================================

# Configuration Variables
# You can modify these to customize your setup
CLUSTER_NAME='topi-dev'
REG_NAME='kind-registry'
REG_PORT='15001'
WORKER_NODES=3  # Number of worker nodes (modify as needed)

# Port Mappings (modify or add more as needed)
# Format: containerPort:hostPort
MINIO_API_PORT='30900:9000'      # MinIO API
MINIO_CONSOLE_PORT='30901:9001'  # MinIO Console

echo "🚀 Setting up Kind cluster: ${CLUSTER_NAME}"
echo "   - Registry: ${REG_NAME}:${REG_PORT}"
echo "   - Worker nodes: ${WORKER_NODES}"
echo ""

# =============================================================================
# 1. Create registry container unless it already exists
# =============================================================================
echo "📦 Setting up local Docker registry..."
if [ "$(docker inspect -f '{{.State.Running}}' "${REG_NAME}" 2>/dev/null || true)" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:${REG_PORT}:5000" --network bridge --name "${REG_NAME}" \
    registry:2
  echo "   ✅ Registry created: ${REG_NAME}"
else
  echo "   ℹ️  Registry already running: ${REG_NAME}"
fi

# =============================================================================
# 2. Create kind cluster with multi-node setup and port mappings
# =============================================================================
echo ""
echo "🏗️  Creating Kind cluster with multi-node setup..."

# Build the worker nodes section dynamically
WORKER_NODES_CONFIG=""
for i in $(seq 1 ${WORKER_NODES}); do
  WORKER_NODES_CONFIG="${WORKER_NODES_CONFIG}- role: worker
"
done

cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
- role: control-plane
  extraPortMappings:
  # MinIO API - modify port mappings here as needed
  - containerPort: $(echo ${MINIO_API_PORT} | cut -d: -f1)
    hostPort: $(echo ${MINIO_API_PORT} | cut -d: -f2)
    protocol: TCP
  # MinIO Console
  - containerPort: $(echo ${MINIO_CONSOLE_PORT} | cut -d: -f1)
    hostPort: $(echo ${MINIO_CONSOLE_PORT} | cut -d: -f2)
    protocol: TCP
${WORKER_NODES_CONFIG}
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
EOF

# =============================================================================
# 3. Configure registry on all cluster nodes
# =============================================================================
# This is necessary because localhost resolves to loopback addresses that are
# network-namespace local. We tell containerd to alias localhost:REG_PORT to
# the registry container when pulling images.
echo ""
echo "🔧 Configuring registry on cluster nodes..."
REGISTRY_DIR="/etc/containerd/certs.d/localhost:${REG_PORT}"
for node in $(kind get nodes --name ${CLUSTER_NAME}); do
  docker exec "${node}" mkdir -p "${REGISTRY_DIR}"
  cat <<EOF | docker exec -i "${node}" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://${REG_NAME}:5000"]
EOF
  echo "   ✅ Configured registry on node: ${node}"
done

# =============================================================================
# 4. Connect registry to cluster network
# =============================================================================
echo ""
echo "🔌 Connecting registry to cluster network..."
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REG_NAME}")" = 'null' ]; then
  docker network connect "kind" "${REG_NAME}"
  echo "   ✅ Registry connected to Kind network"
else
  echo "   ℹ️  Registry already connected to Kind network"
fi

# =============================================================================
# 5. Document the local registry
# =============================================================================
# This creates a ConfigMap to document the registry location for tools
# See: https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
echo ""
echo "📝 Documenting local registry..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REG_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

# =============================================================================
# Setup Complete!
# =============================================================================
echo ""
echo "✨ ============================================== ✨"
echo "   Kind cluster setup complete!"
echo "✨ ============================================== ✨"
echo ""
echo "📋 Cluster Information:"
echo "   Cluster name: ${CLUSTER_NAME}"
echo "   Registry:     localhost:${REG_PORT}"
echo "   Nodes:        1 control-plane + ${WORKER_NODES} workers"
echo ""
echo "🔌 Port Mappings:"
echo "   MinIO API:     localhost:$(echo ${MINIO_API_PORT} | cut -d: -f2) -> ${MINIO_API_PORT}"
echo "   MinIO Console: localhost:$(echo ${MINIO_CONSOLE_PORT} | cut -d: -f2) -> ${MINIO_CONSOLE_PORT}"
echo ""
echo "📝 Useful Commands:"
echo "   kubectl cluster-info --context kind-${CLUSTER_NAME}"
echo "   kubectl get nodes"
echo "   kind delete cluster --name ${CLUSTER_NAME}"
echo ""
echo "🎯 Next Steps:"
echo "   - Build and push images: make kind-push-all"
echo "   - Deploy MinIO: kubectl apply -f <minio-manifests>"
echo "   - Deploy Topi: make kind-deploy-scheduler"
echo ""
