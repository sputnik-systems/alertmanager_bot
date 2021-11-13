# Development
You may use [docker runtime](https://docs.docker.com/engine/install/) and [minikube](https://kubernetes.io/ru/docs/tasks/tools/install-minikube/) for local development and testing.

## Run project
<br>Building and running project:
```
make LOCAL_BOT_TOKEN="my-bot-token" local-deploy
```
Re-deploy project:
```
make LOCAL_BOT_TOKEN="my-bot-token" local-deploy
```
Cleaning environment:
```
make clean
```

## Grafana
Now grafana will be installed at enviromnent init step(`make minikube`) with user/pass `admin/admin`.
<br>You can use kubectl port-forward for accessing to it:
```
kubectl port-forward svc/grafana 8080:80
```
