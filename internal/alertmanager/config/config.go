package config

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"sync"

	amcfg "github.com/prometheus/alertmanager/config"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	key types.NamespacedName
	wh  []*amcfg.WebhookConfig
	kc  client.Client
	mux *sync.Mutex
}

func New(key types.NamespacedName, wu *url.URL, kc client.Client) *Config {
	wc := &amcfg.WebhookConfig{
		NotifierConfig: amcfg.NotifierConfig{
			VSendResolved: true,
		},
		URL: &amcfg.URL{
			URL: wu,
		},
	}
	wh := []*amcfg.WebhookConfig{wc}

	return &Config{
		key: key,
		wh:  wh,
		kc:  kc,
		mux: &sync.Mutex{},
	}
}

func (c *Config) RegisterReceiver(receiver int64) error {
	conf, err := c.Get()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	err = c.addReceiver(conf, receiver)
	if err != nil {
		return fmt.Errorf("failed to add receiver: %s", err)
	}

	err = c.write(conf)
	if err != nil {
		return fmt.Errorf("failed to save alertmanger config: %s", err)
	}

	return nil
}

func (c *Config) DisableReceiver(receiver int64) error {
	conf, err := c.Get()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	err = delAllRoutes(conf, receiver)
	if err != nil {
		return fmt.Errorf("failed deleting all routes for given receiver: %s", err)
	}

	err = removeReceiver(conf, receiver)
	if err != nil {
		return fmt.Errorf("failed to remove receiver from config: %s", err)
	}

	err = c.write(conf)
	if err != nil {
		return fmt.Errorf("failed to save alertmanger config: %s", err)
	}

	return nil
}

func (c *Config) IsReceiverExists(receiver int64) (bool, error) {
	conf, err := c.Get()
	if err != nil {
		return false, fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	r := strconv.FormatInt(receiver, 10)
	if p := getReceiverPosition(conf, r); p == -1 {
		return false, nil
	}

	return true, nil
}

func (c *Config) AddRoute(receiver int64, group string) error {
	conf, err := c.Get()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	r := strconv.FormatInt(receiver, 10)
	match := make(map[string]string)
	match["alertgroup"] = group

	p := getRoutePosition(conf.Route.Routes, r, match)
	if p != -1 {
		log.Printf("route already exists: %s/%s", r, group)

		return nil
	}

	route := &amcfg.Route{
		Receiver: r,
		Continue: true,
		Match:    match,
	}

	conf.Route.Routes = append(conf.Route.Routes, route)

	err = c.write(conf)
	if err != nil {
		return fmt.Errorf("failed to save alertmanger config: %s", err)
	}

	return nil
}

func (c *Config) RemoveRoute(receiver int64, group string) error {
	conf, err := c.Get()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	r := strconv.FormatInt(receiver, 10)
	match := make(map[string]string)
	match["alertgroup"] = group

	p := getRoutePosition(conf.Route.Routes, r, match)
	if p == -1 {
		log.Printf("receiver doesn't have routes now: %s/%s", r, group)

		return nil
	}

	conf.Route.Routes[p] = conf.Route.Routes[len(conf.Route.Routes)-1]
	conf.Route.Routes = conf.Route.Routes[:len(conf.Route.Routes)-1]

	err = c.write(conf)
	if err != nil {
		return fmt.Errorf("failed to save alertmanger config: %s", err)
	}

	return nil
}

func (c *Config) getSecret() (*v1.Secret, error) {
	secret := &v1.Secret{}

	err := c.kc.Get(context.Background(), c.key, secret)
	if err != nil {
		return nil, err
	}

	if _, ok := secret.Data["alertmanager.yaml"]; !ok {
		return nil, fmt.Errorf("secret not contain alertmanager.yaml file")
	}

	return secret, nil
}

func (c *Config) Get() (*amcfg.Config, error) {
	secret, err := c.getSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret with alertmanager config: %s", err)
	}

	conf, err := amcfg.Load(string(secret.Data["alertmanager.yaml"]))
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal alertmanager.yaml file: %s", err)
	}

	return conf, nil
}

func (c *Config) write(conf *amcfg.Config) error {
	secret, err := c.getSecret()
	if err != nil {
		return fmt.Errorf("failed to get secret with alertmanager config: %s", err)
	}

	data := conf.String()
	secret.Data["alertmanager.yaml"] = []byte(data)

	err = c.kc.Update(context.Background(), secret)
	if err != nil {
		return fmt.Errorf("failed to update alertmanager secret with config: %s", err)
	}

	return nil
}

func (c *Config) addReceiver(conf *amcfg.Config, receiver int64) error {
	r := strconv.FormatInt(receiver, 10)
	if pos := getReceiverPosition(conf, r); pos == -1 {
		rc := &amcfg.Receiver{
			Name:           r,
			WebhookConfigs: c.wh,
		}
		conf.Receivers = append(conf.Receivers, rc)
	} else {
		conf.Receivers[pos].WebhookConfigs = c.wh
	}

	return nil
}
