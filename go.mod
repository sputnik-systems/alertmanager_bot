module github.com/sputnik-systems/alertmanager_bot

go 1.16

require (
	github.com/VictoriaMetrics/operator v0.9.1
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/prometheus/alertmanager v0.21.1-0.20200911160112-1fdff6b3f939
	github.com/prometheus/common v0.15.0
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/spf13/afero v1.3.4 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/vcraescu/go-paginator/v2 v2.0.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/net v0.0.0-20210220033124-5f55cee0dc0d // indirect
	golang.org/x/oauth2 v0.0.0-20210220000619-9bb904979d93 // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.5 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	gopkg.in/tucnak/telebot.v3 v3.0.0-20211105204051-d2269534fa9b
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.1
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.3.1
	k8s.io/api => k8s.io/api v0.18.16
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.16
	k8s.io/client-go => k8s.io/client-go v0.18.16
)
