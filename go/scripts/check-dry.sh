#!/usr/bin/env bash

set -euo pipefail

go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .

