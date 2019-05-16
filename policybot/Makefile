VERSION := $(shell date +%Y%m%d%H%M%S)
IMG := gcr.io/istio-testing/policybot:$(VERSION)

deploy: build_docker
	@docker push $(IMG)
	@gcloud beta run deploy policybot --image $(IMG)
	@echo "Pushed and deployed image $(IMG)"

build_docker:
	@CGO_ENABLED=0 GOOS=linux go build
	@docker build --no-cache --quiet --tag $(IMG) .
