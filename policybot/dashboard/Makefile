ISTIO_SERVE_DOMAIN ?= localhost
export ISTIO_SERVE_DOMAIN

img := gcr.io/istio-testing/website-builder:2019-05-03
docker := docker run -e INTERNAL_ONLY=true -t -i --sig-proxy=true --rm -v $(shell pwd):/site -w /site $(img)

ifeq ($(INTERNAL_ONLY),)
docker := docker run -t -i --sig-proxy=true --rm -v $(shell pwd):/site -w /site $(img)
endif

build:
	@$(docker) scripts/build_site.sh

opt:
	@$(docker) scripts/opt_site.sh

lint:
	@$(docker) scripts/lint_site.sh

clean:
	@rm -fr tmp
