.PHONY: run-dashboard e2e test

run-dashboard:
	@./hack/run-dashboard.sh

e2e:
	@./hack/e2e.sh

test:
	@echo "Running unit tests..."
	@go test $(shell go list ./... | grep -v '/test/') -v

