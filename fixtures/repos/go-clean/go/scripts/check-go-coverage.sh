#!/usr/bin/env bash
set -euo pipefail

go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
