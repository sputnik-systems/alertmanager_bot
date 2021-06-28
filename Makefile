KIND := kind
KIND_CLUSTER_NAME := alertmanager-bot
KUBECTL := kubectl
HELM := helm
DOCKER_REGISTRY := k8s-registry.sputnik.systems/tools/alertmanager-bot
DOCKER_IMAGE_TAG := $(shell git describe --tags)
DOCKER_LOCAL_IMAGE_TAG := $(shell git describe --tags)-$(shell date +%s)
LOCAL_BOT_TOKEN := 1668372653:AAH9XvKn3y6iLaSXyrAl1svy8T2aI5Z-STA
LOCAL_REGISTRATION_TOKEN := d4IRGKuCNJohuhCk7w0x9esfEItTgm98

.PHONY: kind
kind: # init kind cluster if it does not exists and switch kubeconfig context
		${KIND} get clusters | grep ${KIND_CLUSTER_NAME} || ${KIND} create cluster --name ${KIND_CLUSTER_NAME}
		${KUBECTL} config use-context kind-${KIND_CLUSTER_NAME}
		${KUBECTL} kustomize deployments/kustomize/vm-operator | ${KUBECTL} apply -f -
		${KUBECTL} kustomize deployments/kustomize | ${KUBECTL} apply -f -
		${HELM} repo add grafana https://grafana.github.io/helm-charts
		${HELM} upgrade --install --values deployments/helm.grafana.values.yaml grafana grafana/grafana

.PHONY: build-image
build-image: # build docker image
		docker build -t ${DOCKER_REGISTRY}:${DOCKER_IMAGE_TAG} .

.PHONY: build-local-image
build-local-image: # build docker image
		docker build -t ${DOCKER_REGISTRY}:${DOCKER_LOCAL_IMAGE_TAG} .

.PHONY: local-deploy
local-deploy:
		${KIND} load docker-image --name ${KIND_CLUSTER_NAME} ${DOCKER_REGISTRY}:${DOCKER_LOCAL_IMAGE_TAG}
		${HELM} upgrade --install --set bot_token="${LOCAL_BOT_TOKEN}",user_register_token="${LOCAL_REGISTRATION_TOKEN}",werf.image.bot="${DOCKER_REGISTRY}:${DOCKER_LOCAL_IMAGE_TAG}" alertmanager-bot ./deployments/helm-chart

.PHONY: build
build: build-image

.PHONY: local
local: kind build-local-image local-deploy

.PHONY: clean
clean:
		${KIND} delete cluster --name ${KIND_CLUSTER_NAME}
