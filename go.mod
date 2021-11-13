module github.com/sputnik-systems/alertmanager_bot

go 1.16

replace (
	k8s.io/api => k8s.io/api v0.22.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.3
	k8s.io/client-go => k8s.io/client-go v0.22.3
)

require (
	github.com/VictoriaMetrics/operator v0.20.3
	github.com/coreos/go-oidc/v3 v3.1.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.52.0
	github.com/prometheus/alertmanager v0.23.0
	github.com/prometheus/common v0.32.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.9.0
	github.com/vcraescu/go-paginator/v2 v2.0.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	gopkg.in/tucnak/telebot.v3 v3.0.0-20211108093419-844466d6faf3
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.10.3
)
