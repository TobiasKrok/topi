#!/bin/bash
# test-local.sh - Run scheduler locally for quick testing

set -e

CLUSTER_NAME=${CLUSTER_NAME:-topi-dev}

echo "🔍 Checking if cluster exists..."
if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "❌ Cluster ${CLUSTER_NAME} not found. Run '../local-dev-cycle.sh setup' first"
    exit 1
fi

echo "📦 Installing CRDs..."
make install

echo "🚀 Running controller locally..."
echo ""
echo "💡 TIP: The controller will watch your Kind cluster"
echo "   Create a BuildJob in another terminal with:"
echo "   kubectl apply -f config/samples/"
echo ""

make run
