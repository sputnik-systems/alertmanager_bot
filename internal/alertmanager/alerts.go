// from github.com/metalmatze/alertmanager-bot/pkg/alertmanager/alerts.go
package alertmanager

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/common/model"
)

type alertResponse struct {
	Alerts []*AlertsData `json:"data,omitempty"`
}

type AlertsData struct {
	model.Alert

	Receivers *Receivers `json:"receivers"`
}

type Receivers []string

func (r *Receivers) contains(receiver string) bool {
	for _, value := range *r {
		if value == receiver {
			return true
		}
	}

	return false
}

// ListAlerts returns a slice of Alert and an error.
func (a *Alertmanager) ListAlerts(receiver string, params map[string]string) ([]*model.Alert, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/alerts", a.url),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed make request obj: %s", err)
	}

	query := req.URL.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed make request: %s", err)
	}

	var alertResponse alertResponse
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err := dec.Decode(&alertResponse); err != nil {
		return nil, err
	}

	var alerts []*model.Alert
	for _, value := range alertResponse.Alerts {
		if value.Receivers.contains(receiver) {
			alert := &model.Alert{
				Labels:       value.Labels,
				Annotations:  value.Annotations,
				StartsAt:     value.StartsAt,
				EndsAt:       value.EndsAt,
				GeneratorURL: value.GeneratorURL,
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts, err
}

type alertWebhook struct {
	Receiver string         `json:"receiver"`
	Alerts   []*model.Alert `json:"alerts,omitempty"`
}

func GetWebhookData(r *http.Request) ([]*model.Alert, string, error) {
	var alertWebhook alertWebhook

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&alertWebhook); err != nil {
		return nil, "", fmt.Errorf("failed read webhook request body: %s", err)
	}

	return alertWebhook.Alerts, alertWebhook.Receiver, nil
}
