package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"

	amcfg "github.com/prometheus/alertmanager/config"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrNotFound = errors.New("not found")
)

type Config struct {
	kd, km *types.NamespacedName
	wh     []*amcfg.WebhookConfig
	kc     client.Client
	mux    *sync.Mutex
}

func New(kd, km *types.NamespacedName, wu *url.URL, kc client.Client) *Config {
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
		kd:  kd,
		km:  km,
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

	r := strconv.FormatInt(receiver, 10)
	conf.Route.Routes = removeAllRoutes(conf.Route.Routes, r)

	p := getReceiverPosition(conf.Receivers, r)
	if p == -1 {
		return ErrNotFound
	}
	conf.Receivers[p] = conf.Receivers[len(conf.Receivers)-1]
	conf.Receivers = conf.Receivers[:len(conf.Receivers)-1]

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
	if p := getReceiverPosition(conf.Receivers, r); p == -1 {
		return false, nil
	}

	return true, nil
}

func (c *Config) IsRouteExists(receiver int64, match map[string]string) (bool, error) {
	conf, err := c.Get()
	if err != nil {
		return false, fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	r := strconv.FormatInt(receiver, 10)
	if p := getRoutePosition(conf.Route.Routes, r, match); p == -1 {
		return false, nil
	}

	return true, nil
}

func (c *Config) AddRoute(receiver int64, match map[string]string) error {
	conf, err := c.Get()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	r := strconv.FormatInt(receiver, 10)
	p := getRoutePosition(conf.Route.Routes, r, match)
	if p != -1 {
		log.Printf("route %s with match %v already exists", r, match)

		return nil
	}

	if match == nil {
		conf.Route.Routes = removeAllRoutes(conf.Route.Routes, r)
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

func (c *Config) RemoveRoute(receiver int64, match map[string]string) error {
	conf, err := c.Get()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	r := strconv.FormatInt(receiver, 10)
	p := getRoutePosition(conf.Route.Routes, r, match)
	if p == -1 {
		log.Printf("route %s with match %v doesn't exists", r, match)

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

func (c *Config) FindMatchByPrefix(receiver int64, prefix string) (map[string]string, error) {
	conf, err := c.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	r := strconv.FormatInt(receiver, 10)
	routes := listRoutes(conf.Route.Routes, r)
	for _, value := range routes {
		if group, ok := value.match["alertgroup"]; ok && strings.HasPrefix(group, prefix) {
			return value.match, nil
		}
	}

	return nil, ErrNotFound
}

func (c *Config) Get() (*amcfg.Config, error) {
	sd, err := c.getSecret(c.kd)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret with alertmanager config: %s", err)
	}

	conf, err := amcfg.Load(string(sd.Data["alertmanager.yaml"]))
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal alertmanager.yaml file: %s", err)
	}

	if c.km != nil {
		sm, err := c.getSecret(c.km)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret with alertmanager config: %s", err)
		}

		cm, err := amcfg.Load(string(sm.Data["alertmanager.yaml"]))
		if err != nil {
			return nil, fmt.Errorf("failed unmarshal alertmanager.yaml file: %s", err)
		}

		cm.Route.Routes = conf.Route.Routes
		cm.Receivers = conf.Receivers

		return cm, nil
	}

	return conf, nil
}

func (c *Config) Sync() error {
	conf, err := c.Get()
	if err != nil {
		return fmt.Errorf("failed to sync alertmanager configs: %s", err)
	}

	return c.write(conf)
}

func (c *Config) getSecret(key *types.NamespacedName) (*v1.Secret, error) {
	s := &v1.Secret{}

	if err := c.kc.Get(context.Background(), *key, s); err != nil {
		return nil, err
	}

	if _, ok := s.Data["alertmanager.yaml"]; !ok {
		return nil, fmt.Errorf("secret not contain alertmanager.yaml file")
	}

	return s, nil
}

func (c *Config) write(conf *amcfg.Config) error {
	s, err := c.getSecret(c.kd)
	if err != nil {
		return fmt.Errorf("failed to get secret with alertmanager config: %s", err)
	}

	data := conf.String()
	s.Data["alertmanager.yaml"] = []byte(data)

	err = c.kc.Update(context.Background(), s)
	if err != nil {
		return fmt.Errorf("failed to update alertmanager secret with config: %s", err)
	}

	return nil
}

func (c *Config) addReceiver(conf *amcfg.Config, receiver int64) error {
	r := strconv.FormatInt(receiver, 10)
	if pos := getReceiverPosition(conf.Receivers, r); pos == -1 {
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
