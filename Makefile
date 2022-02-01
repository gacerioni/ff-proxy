ifndef GOPATH
        GOPATH := $(shell go env GOPATH)
endif
ifndef GOBIN
        GOBIN := $(shell go env GOPATH)/bin
endif
ifndef DOCKER_BUILD_OPTS
	DOCKER_BUILD_OPTS := --build
endif

.DEFAULT_GOAL := all

tools = $(addprefix $(GOBIN)/, golangci-lint golint gosec goimports gocov gocov-html)
deps = $(addprefix $(GOBIN)/, oapi-codegen)

dep: $(deps) ## Install the deps required to generate code and build feature flags
	@echo "Installing dependances"
	@go mod download

tools: $(tools) ## Install tools required for the build
	@echo "Installed tools"

all: dep generate build ## Pulls down required deps, runs required code generation and builds the ff-proxy binary

# Install oapi-codegen to generate ff server code from the apis
$(GOBIN)/oapi-codegen:
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.6.0

PHONY+= generate
generate: ## Generates the client for the ff-servers client service
	oapi-codegen -generate client -package=client ./ff-api/docs/release/client-v1.yaml > gen/client/services.gen.go
	oapi-codegen -generate types -package=client ./ff-api/docs/release/client-v1.yaml > gen/client/types.gen.go
	oapi-codegen -generate client -package=admin  ./ff-api/docs/release/admin-v1.yaml > gen/admin/services.gen.go
	oapi-codegen -generate types -package=admin ./ff-api/docs/release/admin-v1.yaml > gen/admin/types.gen.go

PHONY+= build
build: generate ## Builds the ff-proxy service binary
	CGO_ENABLED=0 go build -o ff-proxy ./cmd/ff-proxy/main.go

PHONY+= build-race
build-race: generate ## Builds the ff-proxy service binary with the race detector enabled
	CGO_ENABLED=1 go build -race -o ff-proxy ./cmd/ff-proxy/main.go

image: ## Builds a docker image for the proxy called ff-proxy:latest
	@echo "Building Feature Flag Proxy Image"
	@docker build --build-arg GITHUB_ACCESS_TOKEN=${GITHUB_ACCESS_TOKEN} -t ff-proxy:latest -f ./Dockerfile .

PHONY+= test
test: ## Run the go tests (runs with race detector enabled)
	@echo "Running tests"
	go test -race -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

PHONY+= integration-test
integration-test: ## Brings up pushpin & redis and runs go tests (runs with race detector enabled)
	@echo "Running tests"
	make dev
	go test -short -race -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	make stop


###########################################
# we use -coverpkg command to report coverage for any line run from any package
# the input for this param is a comma separated list of all packages in our repo excluding the /cmd/ and /gen/ directories
###########################################
test-report: ## Run the go tests and generate a coverage report
	@echo "Running tests"
	make dev
	go test -short -covermode=atomic -coverprofile=proxy.cov -coverpkg=$(shell go list ./... | grep -v /cmd/ | grep -v /gen/ | xargs | sed -e 's/ /,/g') ./...
	gocov convert ./proxy.cov | gocov-html > ./proxy_test_coverage.html
	make stop

PHONY+= dev
dev: ## Brings up services that the proxy uses
	docker-compose -f ./docker-compose.yml up -d --remove-orphans redis pushpin

PHONY+= run
run: ## Runs the proxy and redis
	docker-compose --env-file .offline.docker.env -f ./docker-compose.yml up ${DOCKER_BUILD_OPTS} --remove-orphans redis proxy

PHONY+= stop
stop: ## Stops all services brought up by make run
	docker-compose -f ./docker-compose.yml down --remove-orphans

PHONY+= clean-redis
clean-redis: ## Removes all data from redis
	redis-cli -h localhost -p 6379 flushdb

PHONY+= build-example-sdk
build-example-sdk: ## builds an example sdk that can be used for hitting the proxy
	CGO_ENABLED=0 go build -o ff-example-sdk ./cmd/example-sdk/main.go


#########################################
# Checks
# These lint, format and check the code for potential vulnerabilities
#########################################
PHONY+= check
check: lint format sec ## Runs linter, goimports and gosec

PHONY+= lint
lint: tools ## lint the golang code
	@echo "Linting $(1)"
	@golint ./...

PHONY+= tools
format: tools ## Format go code and error if any changes are made
	@echo "Formating ..."
	@goimports -w .
	@echo "Formatting complete"

PHONY+= sec
sec: tools ## Run the security checks
	@echo "Checking for security problems ..."
	@gosec -quiet -confidence high -severity medium ./...
	@echo "No problems found"

###########################################
# Install Tools and deps
#
# These targets specify the full path to where the tool is installed
# If the tool already exists it wont be re-installed.
###########################################

# Install golangci-lint
$(GOBIN)/golangci-lint:
	@echo "🔘 Installing golangci-lint... (`date '+%H:%M:%S'`)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin

# Install golint to lint code
$(GOBIN)/golint:
	@echo "🔘 Installing golint ... (`date '+%H:%M:%S'`)"
	@go install golang.org/x/lint/golint@latest

# Install goimports to format code
$(GOBIN)/goimports:
	@echo "🔘 Installing goimports ... (`date '+%H:%M:%S'`)"
	@go install golang.org/x/tools/cmd/goimports@latest

# Install gocov to parse code coverage
$(GOBIN)/gocov:
	@echo "🔘 Installing gocov ... (`date '+%H:%M:%S'`)"
	@go install github.com/axw/gocov/gocov@latest

# Install gocov-html to generate a code coverage html file
$(GOBIN)/gocov-html:
	@echo "🔘 Installing gocov-html ... (`date '+%H:%M:%S'`)"
	@go install github.com/matm/gocov-html@latest

# Install gosec for security scans
$(GOBIN)/gosec:
	@echo "🔘 Installing gosec ... (`date '+%H:%M:%S'`)"
	@curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(GOPATH)/bin

help: ## show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: $(PHONY)
