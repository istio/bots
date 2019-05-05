VERSION := $(shell date +%Y%m%d%H%M%S)
IMG := gcr.io/istio-testing/policybot:$(VERSION)

deploy: build
	@docker push $(IMG)
	@gcloud beta run deploy policybot --image $(IMG)
	@echo "Pushed and deployed image $(IMG)"

run: build
	docker run -t -i --sig-proxy=true -p 8080:8080 --rm $(IMG)

run_local:
	@go build
	@./policybot server --config ./policybot.yaml

build:
	@CGO_ENABLED=0 GOOS=linux go build
	@docker build --no-cache --quiet --tag $(IMG) .
