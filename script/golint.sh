#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Running Go linting checks..."

# Run golint in Docker
docker run --rm \
  -v "$(pwd):/app" \
  -w /app \
  golang:1.23-alpine \
  sh -c "
    # Install golint
    go install golang.org/x/lint/golint@latest
    
    # Run golint on all Go files
    LINT_RESULTS=\$(find . -type f -name '*.go' | xargs /go/bin/golint)
    
    if [ -n \"\$LINT_RESULTS\" ]; then
      echo -e \"${RED}Linting issues found:${NC}\"
      echo \"\$LINT_RESULTS\"
      exit 1
    else
      echo -e \"${GREEN}No linting issues found.${NC}\"
    fi
    
    # Run go vet for additional static analysis
    echo \"Running go vet...\"
    go vet ./...
    
    # Check for formatting issues
    echo \"Checking code formatting...\"
    GOFMT_RESULTS=\$(gofmt -l .)
    if [ -n \"\$GOFMT_RESULTS\" ]; then
      echo -e \"${RED}Formatting issues found in:${NC}\"
      echo \"\$GOFMT_RESULTS\"
      exit 1
    else
      echo -e \"${GREEN}No formatting issues found.${NC}\"
    fi
  "

echo -e "${GREEN}All linting checks passed!${NC}"
