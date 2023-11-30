-include .env
.EXPORT_ALL_VARIABLES:

.PHONY: all test lint vet fmt travis coverage checkfmt prepare deps build

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m


all: test vet checkfmt

travis: test checkfmt coverage

prepare: fmt test

test:
	@echo "$(OK_COLOR)Test packages$(NO_COLOR)"
	@go test -v ./...

coverage:
	@echo "$(OK_COLOR)Make coverage report$(NO_COLOR)"
	@./script/coverage.sh
	-goveralls -coverprofile=gover.coverprofile -service=travis-ci

checkfmt:
	@echo "$(OK_COLOR)Check formats$(NO_COLOR)"
	@./script/checkfmt.sh .

fmt:
	@echo "$(OK_COLOR)Check fmt$(NO_COLOR)"
	@echo "FIXME go fmt does not format imports, should be fixed"
	@go fmt

tools:
	@echo "$(OK_COLOR)Install tools$(NO_COLOR)"
	go install golang.org/x/tools/cmd/goimports@latest
	go get golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls

deps:
	$(info Install dependencies...)
	go mod tidy
	go mod download

#====================  DOCKER  ====================

IMAGE_NAME ?= denisqsound/specter
PLATFORMS := linux/arm64 linux/amd64 # darwin/amd64 darwin/arm64
IMAGE_FILE := specter# pipeline

define BUILD_template
build-$(1):
	$$(info Build for $(1)...)
	GOOS=$$(word 1,$$(subst /, ,$1)) GOARCH=$$(word 2,$$(subst /, ,$1)) go build -o bin/specter-$$(subst /,-,$1) .
endef

$(foreach platform,$(PLATFORMS),$(eval $(call BUILD_template,$(platform))))

build:
	$(foreach platform,$(PLATFORMS),$(MAKE) build-$(platform);)

define DC_BUILD_template
dc-build-$(1):
	docker buildx build --progress=plain --no-cache --platform $$(subst -,/,$1) --build-arg OS=$$(word 1,$$(subst /, ,$1)) --build-arg ARCH=$$(word 2,$$(subst /, ,$1)) -t $(IMAGE_NAME):$$(subst /,-,$1)-$(IMAGE_FILE) --load -f .wlrm/build/${IMAGE_FILE}.Dockerfile .
endef

$(foreach platform,$(PLATFORMS),$(eval $(call DC_BUILD_template,$(platform))))

dc-build:
	$(foreach platform,$(PLATFORMS),$(MAKE) dc-build-$(platform);)

dc-manifest:
	docker manifest create $(IMAGE_NAME):latest $(foreach platform,$(PLATFORMS),--amend $(IMAGE_NAME):$(subst /,-,$(platform)))
	docker manifest push $(IMAGE_NAME):latest

define DC_PUSH_template
dc-push-$(1):
	docker push $(IMAGE_NAME):$(subst /,-,$(1))-$(IMAGE_FILE)
endef

$(foreach platform,$(PLATFORMS),$(eval $(call DC_PUSH_template,$(platform))))

dc-push:
	$(foreach platform,$(PLATFORMS),$(MAKE) dc-push-$(platform);)

dc-publish:
	$(MAKE) build
	$(MAKE) dc-build
	$(MAKE) dc-push
	$(MAKE) dc-manifest


dc-run:
	docker run -it \
	  -v $(CURDIR)/bin/load.yaml:/app/load.yaml \
	  -v $(CURDIR)/bin/ammo.json:/app/ammo.json \
	  --entrypoint sh \
	  $(IMAGE_NAME):latest

upload:
	./script/load.sh