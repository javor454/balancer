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
.PHONY: up down traffic kill register lint

up: ## Build in docker
	docker compose up --build

down: ## Stop docker
	docker compose down --volumes --remove-orphans

traffic: ## Simulate some traffic to be balanced
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy & \
	curl -H "Authorization: client1" localhost:8080/dummy &

register: ## Register a new server
	curl -i -X POST http://localhost:8080/register -H "Content-Type: application/json" -d '{"name": "client1", "weight": 3}'

list-registered: ## List registered servers
	curl -i http://localhost:8080/register

kill: ## For gracefull shutdown
	docker kill --signal SIGINT balancer

lint: ## Run linting checks
	./script/golint.sh
