#!/usr/bin/env bash
# 建置映像並（可選）推到內網 registry，對齊 tools 下其他專案的 mmhk/* 命名
set -euo pipefail
cd "$(dirname "$0")"

IMAGE="${IMAGE:-mmhk/svn-docker:2.0}"
TAG="${TAG:-${IMAGE##*:}}"

docker build -t "${IMAGE}" .
echo "built ${IMAGE}"

# 例：REGISTRY=docker-registry.demo2.mixmedia.com/mmhk ./build.sh
if [ -n "${REGISTRY:-}" ]; then
  docker tag "${IMAGE}" "${REGISTRY}/svn:${TAG}"
  docker push "${REGISTRY}/svn:${TAG}"
  echo "pushed ${REGISTRY}/svn:${TAG}"
fi
