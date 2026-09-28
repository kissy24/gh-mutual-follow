GOVULNCHECK_VERSION := v1.8.0
GITLEAKS_VERSION := v8.30.1
ACTIONLINT_VERSION := v1.7.12

.PHONY: check test fmt-check vet build security secrets staged-secrets workflow-check release-check hooks

check: fmt-check vet test build

fmt-check:
	@test -z "$$(gofmt -l main.go internal)" || (gofmt -l main.go internal; exit 1)

vet:
	go vet ./...

test:
	go test -race -count=1 -coverprofile=coverage.out ./...

build:
	go build -trimpath -o gh-mutual-follow .

security: secrets
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

secrets:
	go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION) dir --redact --no-banner .

staged-secrets:
	go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION) git --pre-commit --staged --redact --no-banner

workflow-check:
	go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)

release-check:
	bash scripts/build-release.sh v0.0.0-test

hooks:
	pre-commit install
