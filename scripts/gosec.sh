#!/bin/bash

set -o pipefail

case "${1:-}" in
mimic)
    CEPH_TAG="mimic"
    ;;
nautilus)
    CEPH_TAG="nautilus"
    ;;
octopus)
    CEPH_TAG="octopus"
    ;;
*)
    echo " $0 [command]
Available Commands:
  mimic               set ceph tag to mimic for go sec
  nautilus            set ceph tag to nautilus for go sec
  octopus             set ceph tag to octopus for go sec
" >&2
    ;;
esac


if [[ -x "$(command -v gosec)" ]]; then
  # gosec does not support -mod=vendor, so fallback to non-module support and
  # assume all dependencies are available in ./vendor already
  export GO111MODULE=off
  find cmd internal -type d -print0 | xargs --null gosec -tags ${CEPH_TAG}
else
  echo "WARNING: gosec not found, skipping security tests" >&2
fi
