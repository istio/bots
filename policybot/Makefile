VERSION := $(shell date +%Y%m%d%H%M%S)

TAG ?= $(VERSION)
HUB ?= gcr.io/istio-testing/policybot
IMG := $(HUB):$(TAG)

PROJECT ?= istio-testing
CLUSTER ?= policy-bot
ZONE    ?= us-central1-a

export GO111MODULE=on

dockerrun := docker run -t -i --sig-proxy=true --rm -v $(shell pwd):/site -w /site "gcr.io/istio-testing/build-tools:2019-10-11T13-37-52"

gen:
ifneq ($(CI),true)
	@$(dockerrun) scripts/gen_dashboard.sh
else
	@scripts/gen_dashboard.sh
endif
	@go generate ./...

clean:
	@rm -fr policybot tmp generated */*.gen.go */*/*.gen.go */*/*/*.gen.go

deploy: push deploy_only
.PHONY: deploy

deploy_only: get-cluster-credentials
	@helm template --set image=$(IMG) \
		--set GITHUB_WEBHOOK_SECRET=$(GITHUB_WEBHOOK_SECRET) \
		--set GITHUB_TOKEN=$(GITHUB_TOKEN) \
		--set GCP_CREDS=$(GCP_CREDS) \
		--set SENDGRID_APIKEY=$(SENDGRID_APIKEY) \
		--set GITHUB_OAUTH_CLIENT_ID=$(GITHUB_OAUTH_CLIENT_ID) \
		--set GITHUB_OAUTH_CLIENT_SECRET=$(GITHUB_OAUTH_CLIENT_SECRET) \
		deploy/policybot | kubectl apply -f -
	@echo "Deployed policybot:$(IMG) to GKE"
.PHONY: deploy_only

.PHONY: get-cluster-credentials
get-cluster-credentials:
	gcloud container clusters get-credentials "$(CLUSTER)" --project="$(PROJECT)" --zone="$(ZONE)"

container: gen
	@GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build
	@docker build --no-cache --quiet --tag $(IMG) .
.PHONY: container

push: container
	@echo "When deploying, please make sure of the following:"
	@echo ""
	@echo "  1. That latest sources are up to date."
	@echo "  2. That the dashboard is working locally."
	@echo "  3. That eng.istio.io works after deployment."
	@echo ""
ifneq ($(CI),true)
	@read -p "Press enter to continue."
endif
	@docker push $(IMG)
	@echo "Built container and published to $(IMG)"
.PHONY: push
