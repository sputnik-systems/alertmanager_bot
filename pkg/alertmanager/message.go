package alertmanager

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/prometheus/common/model"
)

func GetMessageText(templatesPath, alertmanagerUrl string, receiver int64, alerts []*model.Alert) (string, error) {
	tmpl, err := FromGlobs(templatesPath)
	if err != nil {
		return "", fmt.Errorf("failed to read template files: %s", err)
	}

	tmpl.ExternalURL, err = url.Parse(alertmanagerUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse alertmanager url: %s", err)
	}

	data := tmpl.Data(strconv.FormatInt(receiver, 10), nil, alerts...)
	out, err := tmpl.ExecuteHTMLString(`{{ template "telegram.default" . }}`, data)
	if err != nil {
		return "", fmt.Errorf("failed to apply template: %s", err)
	}

	return out, nil
}
