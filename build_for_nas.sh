#!/usr/bin/env bash
echo "## Building program for a NAS"
GOOS=linux GOARCH=arm GOARM=7 go build
