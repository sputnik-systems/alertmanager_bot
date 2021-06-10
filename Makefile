KIND := kind
KIND_CLUSTER_NAME := alertmanager-bot
KUBECTL := kubectl
HELM := helm
DOCKER_REGISTRY := k8s-registry.sputnik.systems/tools/alertmanager-bot
DOCKER_IMAGE_TAG := $(shell git describe --tags)
LOCAL_BOT_TOKEN := 1668372653:AAH9XvKn3y6iLaSXyrAl1svy8T2aI5Z-STA
LOCAL_REGISTRATION_TOKEN := d4IRGKuCNJohuhCk7w0x9esfEItTgm98

.PHONY: kind
kind: # init kind cluster if it does not exists and switch kubeconfig context
		${KIND} get clusters | grep alertmanager-bot || ${KIND} create cluster --name ${KIND_CLUSTER_NAME}
		${KUBECTL} config use-context kind-alertmanager-bot
		${KUBECTL} apply -f deployments/manifests/namespace.yaml
		${KUBECTL} kustomize https://github.com/VictoriaMetrics/operator/config/crd | ${KUBECTL} apply -f -
		${KUBECTL} kustomize https://github.com/VictoriaMetrics/operator/config/rbac | ${KUBECTL} apply -f -
		${KUBECTL} kustomize https://github.com/VictoriaMetrics/operator/config/manager | ${KUBECTL} apply -f -
		${KUBECTL} apply -f deployments/manifests/vmrules

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
