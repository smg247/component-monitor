.PHONY: run-dashboard e2e test

run-dashboard:
	@./hack/run-dashboard.sh

e2e:
	@./hack/e2e.sh

test:
	@echo "Running unit tests..."
	# Run tests in all packages except the test package, this is where the e2e tests are located
	@go test $(shell go list ./... | grep -v '/test/') -v

