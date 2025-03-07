MIN_MAKE_VERSION := 3.81

# Min version
ifneq ($(MIN_MAKE_VERSION),$(firstword $(sort $(MAKE_VERSION) $(MIN_MAKE_VERSION))))
	$(error GNU Make $(MIN_MAKE_VERSION) or higher required)
endif

##@ Help
.PHONY: help
help: ## Show all available commands (you are looking at it)
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
.PHONY: up down kill lint

up: ## Build in docker
	docker compose up --build

down: ## Stop docker
	docker compose down --volumes --remove-orphans

traffic: ## Simulate some traffic to be balanced
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy & \
	curl localhost:8080/dummy &

kill: ## For gracefull shutdown
	docker kill --signal SIGINT balancer

lint: ## Run linting checks
	./script/golint.sh
