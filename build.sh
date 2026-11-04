#!/usr/bin/env bash
# 建置 v3 的兩個映像（svn / svn-admin），可選推到內網 registry
set -euo pipefail
cd "$(dirname "$0")"

SVN_IMAGE="${SVN_IMAGE:-mmhk/svn:3.0}"
ADMIN_IMAGE="${ADMIN_IMAGE:-mmhk/svn-admin:3.0}"
SVN_TAG="${SVN_TAG:-${SVN_IMAGE##*:}}"
ADMIN_TAG="${ADMIN_TAG:-${ADMIN_IMAGE##*:}}"

docker build -f Dockerfile.svn -t "${SVN_IMAGE}" .
echo "built ${SVN_IMAGE}"

docker build -f Dockerfile.admin -t "${ADMIN_IMAGE}" .
echo "built ${ADMIN_IMAGE}"

# 例：REGISTRY=docker-registry.demo2.mixmedia.com/mmhk ./build.sh
if [ -n "${REGISTRY:-}" ]; then
  docker tag "${SVN_IMAGE}" "${REGISTRY}/svn:${SVN_TAG}"
  docker push "${REGISTRY}/svn:${SVN_TAG}"
  echo "pushed ${REGISTRY}/svn:${SVN_TAG}"

  docker tag "${ADMIN_IMAGE}" "${REGISTRY}/svn-admin:${ADMIN_TAG}"
  docker push "${REGISTRY}/svn-admin:${ADMIN_TAG}"
  echo "pushed ${REGISTRY}/svn-admin:${ADMIN_TAG}"
fi
