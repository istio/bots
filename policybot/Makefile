VERSION := $(shell date +%Y%m%d%H%M%S)
IMG := gcr.io/istio-testing/policybot:$(VERSION)

deploy: push
	@gcloud beta run deploy policybot --image $(IMG) --project istio-testing
	@echo "Deployed container $(IMG) to Cloud Run"

container:
	@GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build
	@docker build --no-cache --quiet --tag $(IMG) .
.PHONY: container

push: container
	@docker push $(IMG)
	@echo "Built container and published to $(IMG)"
