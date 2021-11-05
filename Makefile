MINIKUBE := minikube
MINIKUBE_PROFILE_NAME := alertmanager-bot
MINIKUBE_KUBERNETES_VERSION := 1.18.9
KUBECTL := kubectl
HELM := helm
DOCKER_REGISTRY := k8s-registry.sputnik.systems/tools/alertmanager-bot
DOCKER_IMAGE_TAG := $(shell git describe --tags)
DOCKER_LOCAL_IMAGE_TAG := $(shell git describe --tags)-$(shell date +%s)
LOCAL_BOT_TOKEN := 1668372653:AAH9XvKn3y6iLaSXyrAl1svy8T2aI5Z-STA
LOCAL_REGISTRATION_TOKEN := d4IRGKuCNJohuhCk7w0x9esfEItTgm98

.PHONY: minikube
minikube: # init minikube cluster if it does not exists and switch kubeconfig context
		${MINIKUBE} profile list | grep ${MINIKUBE_PROFILE_NAME} || ${MINIKUBE} start --profile ${MINIKUBE_PROFILE_NAME} --kubernetes-version=${MINIKUBE_KUBERNETES_VERSION}
		${KUBECTL} config use-context ${MINIKUBE_PROFILE_NAME}
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
local-deploy: build-local-image
		${MINIKUBE} image load --profile ${MINIKUBE_PROFILE_NAME} ${DOCKER_REGISTRY}:${DOCKER_LOCAL_IMAGE_TAG}
		${HELM} upgrade --install --set bot_token="${LOCAL_BOT_TOKEN}",user_register_token="${LOCAL_REGISTRATION_TOKEN}",werf.image.bot="${DOCKER_REGISTRY}:${DOCKER_LOCAL_IMAGE_TAG}" alertmanager-bot ./deployments/helm-chart

.PHONY: build
build: build-image

.PHONY: local
local: minikube local-deploy

.PHONY: clean
clean:
		${MINIKUBE} delete --profile ${MINIKUBE_PROFILE_NAME}
