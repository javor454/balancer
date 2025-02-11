.PHONY: run test clean

# Create tmp directory if it doesn't exist
tmp:
	mkdir -p tmp

# Run the application
run:
	go run main.go

# Run tests with coverage
test: clean tmp
	go test -coverprofile=tmp/coverage.out ./...
	go tool cover -html=tmp/coverage.out -o tmp/coverage.html
	@echo "Coverage report generated at tmp/coverage.html"

# Clean temporary files
clean:
	rm -rf tmp/ 