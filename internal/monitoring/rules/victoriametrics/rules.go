package victoriametrics

import (
	"context"
	"log"

	vm "github.com/VictoriaMetrics/operator/api/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	alertrules "github.com/sputnik-systems/alertmanager_bot/internal/monitoring/rules"
)

func init() {
	// hack for support victorimaetrics operator custom resources with kube client
	if err := vm.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}

type rule struct {
	groups []vm.RuleGroup
}

func Rules(c client.Client) []alertrules.Rule {
	out := make([]alertrules.Rule, 0)

	list := &vm.VMRuleList{}
	if err := c.List(context.Background(), list); err != nil {
		log.Printf("failed to list VMRules: %s", err)

		return out
	}

	for _, item := range list.Items {
		out = append(out, &rule{groups: item.Spec.Groups})
	}

	return out
}

func (r *rule) GetGroupNames() []string {
	groups := make([]string, 0)
	for _, group := range r.groups {
		groups = append(groups, group.Name)
	}

	return groups
}
