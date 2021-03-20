package v1beta1

import (
	"context"

	"github.com/VictoriaMetrics/operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	"github.com/sputnik-systems/alertmanager_bot/pkg/kubernetes/scheme"
)

type VMRuleInterface interface {
	List(opts metav1.ListOptions) (*v1beta1.VMRuleList, error)
	Get(name string, options metav1.GetOptions) (*v1beta1.VMRule, error)
	Create(*v1beta1.VMRule) (*v1beta1.VMRule, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type vmruleClient struct {
	restClient rest.Interface
	ns         string
}

func (c *vmruleClient) List(opts metav1.ListOptions) (*v1beta1.VMRuleList, error) {
	result := v1beta1.VMRuleList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("vmrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(context.Background()).
		Into(&result)

	return &result, err
}

func (c *vmruleClient) Get(name string, opts metav1.GetOptions) (*v1beta1.VMRule, error) {
	result := v1beta1.VMRule{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("vmrules").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(context.Background()).
		Into(&result)

	return &result, err
}

func (c *vmruleClient) Create(project *v1beta1.VMRule) (*v1beta1.VMRule, error) {
	result := v1beta1.VMRule{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource("vmrules").
		Body(project).
		Do(context.Background()).
		Into(&result)

	return &result, err
}

func (c *vmruleClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource("vmrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(context.Background())
}
