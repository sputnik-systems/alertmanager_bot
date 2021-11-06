package alertmanager

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/sputnik-systems/alertmanager_bot/internal/alertmanager/config"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Alertmanager struct {
	url, tp string

	*config.Config
}

func New(a, w, tp string, key types.NamespacedName, kc client.Client) (*Alertmanager, error) {
	if _, err := url.Parse(a); err != nil {
		return nil, fmt.Errorf("given alertmanager url %s is incorrect: %s", a, err)
	}

	wu, err := url.Parse(w)
	if err != nil {
		return nil, fmt.Errorf("given webhook url %s is incorrect: %s", w, err)
	}

	c := config.New(key, wu, kc)

	return &Alertmanager{url: a, tp: tp, Config: c}, nil
}

func (a *Alertmanager) Reload() (*http.Response, error) {
	resp, err := http.Post(
		fmt.Sprintf("%s/-/reload", a.url),
		"application/x-www-form-urlencoded",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed reload alertmanager: %s", err)
	}

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, fmt.Errorf("response body read failed: %s", err)
		}

		return resp, fmt.Errorf("failed alertmanager reload with status code \"%d\" and body \"%s\"", resp.StatusCode, body)
	}

	return resp, nil
}
