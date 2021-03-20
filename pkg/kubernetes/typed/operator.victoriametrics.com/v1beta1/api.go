package v1beta1

import (
	"github.com/VictoriaMetrics/operator/api/v1beta1"
	"k8s.io/client-go/rest"

	"github.com/sputnik-systems/alertmanager_bot/pkg/kubernetes/scheme"
)

type VMV1Beta1Interface interface {
	VMRules(namespace string) VMRuleInterface
}

type VMV1Beta1Client struct {
	restClient rest.Interface
}

func (c *VMV1Beta1Client) VMRules(namespace string) VMRuleInterface {
	return &vmruleClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

// NewForConfig creates a new VMV1Beta1Client for the given config.
func NewForConfig(c *rest.Config) (*VMV1Beta1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &VMV1Beta1Client{client}, nil
}

// NewForConfigOrDie creates a new VMV1Beta1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *VMV1Beta1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new VMV1Beta1Client for the given RESTClient.
func New(c rest.Interface) *VMV1Beta1Client {
	return &VMV1Beta1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1beta1.GroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *VMV1Beta1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
