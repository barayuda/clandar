#!/bin/bash
# Phase 2 build verification — run from the project root.
set -e
go build ./... && echo "go build ./... PASSED"
