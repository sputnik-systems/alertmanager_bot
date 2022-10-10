package alertmanager

import (
	"fmt"
	"net/url"

	"github.com/prometheus/common/model"
)

func (a *Alertmanager) GetMessageText(alerts []*model.Alert) (string, error) {
	tmpl, err := FromGlobs(a.tp)
	if err != nil {
		return "", fmt.Errorf("failed to read template files: %s", err)
	}

	tmpl.ExternalURL, err = url.Parse(a.url)
	if err != nil {
		return "", fmt.Errorf("failed to parse alertmanager url: %s", err)
	}

	data := tmpl.Data(nil, alerts...)
	out, err := tmpl.ExecuteHTMLString(`{{ template "telegram.default" . }}`, data)
	if err != nil {
		return "", fmt.Errorf("failed to apply template: %s", err)
	}

	return out, nil
}
