# Подготовка

Предварительно нужно создать `.env` файл с переменными окружения:
```
BOT_TOKEN=your_bot_token
```
В данном случае запускаются 4 сервиса: `kube-apiserver`, `etcd`, `alertmanager`, `bot`. `bot` - сам сервис с ботом, `kube-apiserver`+`etcd` - api kubernetes. В данной реализации алерты в `alertmanager` следует отправлять вручную(см. ниже).

# Запуск

Нужно поставить [docker-compose](https://docs.docker.com/compose/install/), если используете `docker` в качестве рантайма, или [podman-compose](https://github.com/containers/podman-compose) для работы через [podman](https://github.com/containers/podman). Для варианта с `podman` во всех командах следует заменить `docker-compose` на `podman-compose`
Для сборки контейнера из исходников:
```
$ docker-compose build
```
Запуск контейнеров:
```
$ docker-compose up -d
```
Для остановки всех контейнеров:
```
$ docker-compose down
```

# Алерты
Отправка алертов в `alertmanager`:
```
$ curl -XPOST http://localhost:9093/api/v1/alerts -d '[
  {
    "labels": {
      "alertgroup": "node-exporter1",
      "alertname": "NodeFilesystemAlmostOutOfSpace",
      "device": "/dev/sda",
      "fstype": "ext4",
      "instance": "video.sputnik.systems:9100",
      "job": "node-exporter",
      "mountpoint": "/mnt/69e122be-00c4-450a-a049-740a6fcced6c",
      "prometheus": "monitoring/external",
      "severity": "critical",
      "vmagent": "external"
    },
    "annotations": {
      "description": "Filesystem on /dev/sda at video7.sputnik.systems:9100 has only 2.44% available space left.",
      "runbook_url": "https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/runbook.md#alert-name-nodefilesystemalmostoutofspace",
      "summary": "Filesystem has less than 3% space left."
    },
    "startsAt": "2021-03-04T04:48:01.207293268Z",
    "endsAt": "2022-03-04T18:36:01.21095913Z",
    "generatorURL": "http://vmalert-default-5c4b8bdfc9-9dvk6:8080/api/v1/16447341981611736902/12309954303353466824/status"
  }
]'
```
Просмотр списка алертов:
```
$ curl http://localhost:9093/api/v1/alerts | jq
```
