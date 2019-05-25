VERSION := $(shell date +%Y%m%d%H%M%S)
HUB ?= gcr.io/istio-testing
IMG := $(HUB)/policybot:$(VERSION)

deploy: container
	@gcloud beta run deploy policybot --image $(IMG) --project istio-testing
	@echo "Deployed container $(IMG) to Cloud Run"

container:
	@CGO_ENABLED=0 GOOS=linux go build
	@docker build --no-cache --quiet --tag $(IMG) .
	@docker push $(IMG)
	@echo "Built container and published to $(IMG)"
