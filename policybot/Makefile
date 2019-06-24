VERSION := $(shell date +%Y%m%d%H%M%S)

TAG ?= $(VERSION)
HUB ?= gcr.io/istio-testing/policybot
IMG := $(HUB):$(TAG)
export GO111MODULE=on

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

container:
	@GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build
	@docker build --no-cache --quiet --tag $(IMG) .
.PHONY: container

push: container
	@docker push $(IMG)
	@echo "Built container and published to $(IMG)"
.PHONY: push
