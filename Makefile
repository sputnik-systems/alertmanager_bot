KIND := kind
KIND_CLUSTER_NAME := alertmanager-bot
KUBECTL := kubectl
HELM := helm
DOCKER_REGISTRY := sputniksystemsorg/alertmanager-bot
DOCKER_IMAGE_TAG := $(shell git describe --tags)
LOCAL_BOT_TOKEN :=
LOCAL_REGISTRATION_TOKEN :=

.PHONY: kind
kind: # init kind cluster if it does not exists and switch kubeconfig context
		${KIND} get clusters | grep alertmanager-bot || ${KIND} create cluster --name ${KIND_CLUSTER_NAME}
		${KUBECTL} config use-context kind-alertmanager-bot
		${KUBECTL} kustomize deployments/kustomize/vm-operator | ${KUBECTL} apply -f -
		${KUBECTL} kustomize deployments/kustomize/vmrules | ${KUBECTL} apply -f -

.PHONY: build-image
build-image: # build docker image
		docker build -t ${DOCKER_REGISTRY}:${DOCKER_IMAGE_TAG} .

.PHONY: local-deploy
local-deploy:
		${KIND} load docker-image --name ${KIND_CLUSTER_NAME} ${DOCKER_REGISTRY}:${DOCKER_IMAGE_TAG}
		${HELM} upgrade --install --set bot_token="${LOCAL_BOT_TOKEN}",user_register_token="${LOCAL_REGISTRATION_TOKEN}",werf.image.bot="${DOCKER_REGISTRY}:${DOCKER_IMAGE_TAG}" alertmanager-bot ./deployments/helm-chart

.PHONY: build
build: build-image

.PHONY: local
local: kind build-image local-deploy

.PHONY: clean
clean:
		${KIND} delete cluster --name ${KIND_CLUSTER_NAME}
