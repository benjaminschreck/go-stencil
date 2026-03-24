#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "Usage: $0 vMAJOR.MINOR.PATCH [--push]" >&2
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 1
fi

tag="$1"
push_tag="${2:-}"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "tag must match vMAJOR.MINOR.PATCH" >&2
  exit 1
fi

if [[ -n "$(git status --short)" ]]; then
  echo "working tree must be clean before creating a release tag" >&2
  exit 1
fi

if git rev-parse "${tag}" >/dev/null 2>&1; then
  echo "tag ${tag} already exists locally" >&2
  exit 1
fi

if git ls-remote --tags origin "refs/tags/${tag}" | grep -q .; then
  echo "tag ${tag} already exists on origin" >&2
  exit 1
fi

git tag -a "${tag}" -m "Release ${tag}"
echo "created annotated tag ${tag}"

if [[ "${push_tag}" == "--push" ]]; then
  git push origin "${tag}"
  echo "pushed ${tag} to origin"
elif [[ -n "${push_tag}" ]]; then
  usage
  exit 1
else
  echo "run '$0 ${tag} --push' to publish the release"
fi
