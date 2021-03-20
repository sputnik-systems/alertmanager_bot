#!/bin/bash

set -xe

export KUBECONFIG=/etc/kubernetes/admin.conf

/usr/local/bin/kubectl apply -f https://github.com/VictoriaMetrics/operator/raw/master/config/crd/bases/operator.victoriametrics.com_vmrules.yaml
/usr/local/bin/kubectl apply -f /app/vmrules
/usr/local/bin/kubectl apply -f /app/secrets

/usr/local/bin/alertmanager_bot $@
