GO ?= $(shell which go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)

IMAGE_NAME := niklas-letz/cert-manager-webhook-solidserver
IMAGE_TAG := latest

OUT := $(shell pwd)/_out
LOCALBIN := $(shell pwd)/_test

KUBEBUILDER_VERSION=1.35.0

HELM_FILES := $(shell find deploy/cert-manager-webhook-solidserver)

$(OUT) $(LOCALBIN):
	mkdir -p $@

.PHONY: test
test: $(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/etcd \
			$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kube-apiserver \
			$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kubectl
	TEST_ASSET_ETCD=$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/etcd \
	TEST_ASSET_KUBE_APISERVER=$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kube-apiserver \
	TEST_ASSET_KUBECTL=$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kubectl \
	$(GO) test -v .

$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/etcd \
$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kube-apiserver \
$(LOCALBIN)/k8s/$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kubectl: \
    $(LOCALBIN)/setup-envtest | $(LOCALBIN)
	$(LOCALBIN)/setup-envtest use $(KUBEBUILDER_VERSION) --bin-dir $(LOCALBIN) -p path

$(LOCALBIN)/setup-envtest: | $(LOCALBIN)
	GOBIN=$(LOCALBIN) $(GO) install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: docker-build
docker-build:
	docker build --platform linux/amd64 -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: podman-build
podman-build:
	podman build --platform linux/amd64 -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: docker-build-fast
docker-build-fast: $(OUT)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(OUT)/webhook -ldflags '-w -extldflags "-static"' .
	docker build --platform linux/amd64 -f Dockerfile.fast -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: podman-build-fast
podman-build-fast: $(OUT)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(OUT)/webhook -ldflags '-w -extldflags "-static"' .
	podman build --platform linux/amd64 -f Dockerfile.fast -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: docker-push
docker-push:
	docker push "$(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: podman-push
podman-push:
	podman push "$(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: render-helm
render-helm: $(OUT)
	helm template \
	  cert-manager-webhook-solidserver \
			--set image.repository=$(IMAGE_NAME) \
			--set image.tag=$(IMAGE_TAG) \
			--namespace cert-manager \
			deploy/cert-manager-webhook-solidserver > "$(OUT)/rendered-manifest.yaml"

.PHONY: clean
clean:
	chmod -R u+w $(LOCALBIN) $(OUT) 2>/dev/null || true
	rm -rf $(LOCALBIN) $(OUT)
