version := $(shell git describe --tags)
gitHash := $(shell git rev-parse --short HEAD)
buildDate := $(shell TZ=UTC date +"%Y-%m-%dT%H:%M:%SZ")
pkg :=github.com/runar-rkmedia/skiver-cli/cmd
ldflags=-X '$(pkg).version=$(version)' -X '$(pkg).date=$(buildDate)' -X '$(pkg).commit=$(gitHash)'
usage:
	@echo ""
release:
	LDFLAG="${ldflags}" goreleaser release --rm-dist
snapshot:
	@echo ldflags
	LDFLAG="${ldflags}" goreleaser release --rm-dist --snapshot
readme:
	go run templating/main.go
readme-watch:
	reflex -G 'README.md' -- sh -c "go run templating/main.go"
