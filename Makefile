MINIKUBE := minikube
MINIKUBE_PROFILE_NAME := alertmanager-bot
MINIKUBE_KUBERNETES_VERSION := 1.18.9
KUBECTL := kubectl
HELM := helm
DOCKER_REGISTRY := sputniksystemsorg/alertmanager-bot
DOCKER_IMAGE_TAG := $(shell git describe --tags)
DOCKER_LOCAL_IMAGE_TAG := $(shell git describe --tags)-$(shell date +%s)
LOCAL_BOT_TOKEN :=
LOCAL_BOT_PUBLIC_URL := http://example.org:8080
LOCAL_OIDC_ENABLED := false
LOCAL_OIDC_ISSUER_URL :=
LOCAL_OIDC_CLIENT_ID :=
LOCAL_OIDC_CLIENT_SECRET :=

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
		${HELM} upgrade --install --set bot.token="${LOCAL_BOT_TOKEN}",bot.publicURL="${LOCAL_BOT_PUBLIC_URL}",oidc.enabled="${LOCAL_OIDC_ENABLED}",oidc.issuerURL="${LOCAL_OIDC_ISSUER_URL}",oidc.clientID="${LOCAL_OIDC_CLIENT_ID}",oidc.clientSecret="${LOCAL_OIDC_CLIENT_SECRET}",image.repository="${DOCKER_REGISTRY}",image.tag="${DOCKER_LOCAL_IMAGE_TAG}" alertmanager-bot ./deployments/helm-chart

.PHONY: build
build: build-image

.PHONY: local
local: minikube local-deploy

.PHONY: clean
clean:
		${MINIKUBE} delete --profile ${MINIKUBE_PROFILE_NAME}
