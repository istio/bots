VERSION := $(shell date +%Y%m%d%H%M%S)

TAG ?= $(VERSION)
HUB ?= gcr.io/istio-testing/policybot
IMG := $(HUB):$(TAG)
export GO111MODULE=on

dockerrun := docker run -t -i --sig-proxy=true --rm -v $(shell pwd):/site -w /site "gcr.io/istio-testing/website-builder:2019-05-03"

deploy: push deploy_only
.PHONY: deploy

deploy_only:
	@helm template --set image=$(IMG) \
		--set GITHUB_WEBHOOK_SECRET=$(GITHUB_WEBHOOK_SECRET) \
		--set GITHUB_TOKEN=$(GITHUB_TOKEN) \
		--set ZENHUB_TOKEN=$(ZENHUB_TOKEN) \
		--set GCP_CREDS=$(GCP_CREDS) \
		--set SENDGRID_APIKEY=$(SENDGRID_APIKEY) \
		--set GITHUB_OAUTH_CLIENT_ID=$(GITHUB_OAUTH_CLIENT_ID) \
		--set GITHUB_OAUTH_CLIENT_SECRET=$(GITHUB_OAUTH_CLIENT_SECRET) \
		deploy/kube | kubectl apply -f -
	@echo "Deployed policybot:$(IMG) to GKE"
.PHONY: deploy_only

container: gen
	@GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build
	@docker build --no-cache --quiet --tag $(IMG) .
.PHONY: container

push: container
	@docker push $(IMG)
	@echo "Built container and published to $(IMG)"
.PHONY: push

gen:
	@$(dockerrun) scripts/gen_dashboard.sh
	@go generate ./...

lint:
	@$(dockerrun) scripts/lint_dashboard.sh

clean:
	@rm -fr policybot tmp generated */*.gen.go */*/*.gen.go */*/*/*.gen.go
