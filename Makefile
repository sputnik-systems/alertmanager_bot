KIND := kind
KUBECTL: kubectl
DOCKER_REGISTRY := k8s-registry.spuntik.systems/tools/alertmanager-bot

.PHONY: kind
kind: # init kind cluster if it does not exists and switch kubeconfig context
	${KIND} get clusters | grep alertmanager-bot || ${KIND} create cluster --name alertmanager-bot
	${KUBECTL} config use-context kind-alertmanager-bot

.PHONE: build-image
build-image: # build docker image
	docker build -t ${DOCKER_REGISTRY}:`git
