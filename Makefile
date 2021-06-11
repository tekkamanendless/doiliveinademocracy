all:

googleCloudProject = $(shell cat config.json | jq -r .project)

# `make help` will list all of the targets in this Makefile.
# See: https://stackoverflow.com/questions/4219255/how-do-you-get-the-list-of-targets-in-a-makefile
.PHONY: help
help:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

.PHONY: test
test: test_go test_json

.PHONY: test_go
test_go:
	@echo "Testing Go..."
	@go test ./...

.PHONY: test_json
test_json:
	@echo "Testing JSON..."
	@find . -iname '*.json' | sort --version-sort | while read file; do echo -n "$$file: "; cat "$$file" | jq . >/dev/null 2>&1 && echo "okay" || echo "fail"; done;

.PHONY: run
run:
	go run cmd/server/main.go

.PHONY: deploy
deploy: test
	gcloud functions deploy doiliveinademocracy --runtime go113 --trigger-http --entry-point CloudFunction --allow-unauthenticated --update-env-vars GOOGLE_CLOUD_PROJECT=$(googleCloudProject) --project $(googleCloudProject)

.PHONY: what
what:
	@echo $(googleCloudProject)

