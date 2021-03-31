package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/spf13/viper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"

	"github.com/prometheus/alertmanager/config"

	"github.com/sputnik-systems/alertmanager_bot/pkg/alertmanager"
)

func addReceiverToConfig(cfg *config.Config, receiver int64) error {
	wu, err := url.Parse(viper.GetString("bot.webhook-url"))
	if err != nil {
		return fmt.Errorf("failed to parse bot webhook url: %s", err)
	}

	wc := &config.WebhookConfig{
		NotifierConfig: config.NotifierConfig{
			VSendResolved: true,
		},
		URL: &config.URL{URL: wu},
	}
	wh := []*config.WebhookConfig{wc}
	r := strconv.FormatInt(receiver, 10)
	if pos := getReceiverPosition(cfg, r); pos == -1 {
		rc := &config.Receiver{
			Name:           r,
			WebhookConfigs: wh,
		}
		cfg.Receivers = append(cfg.Receivers, rc)
	} else {
		cfg.Receivers[pos].WebhookConfigs = wh
	}

	return nil
}

func getReceiverPosition(cfg *config.Config, receiver string) int64 {
	for index, value := range cfg.Receivers {
		if value.Name == receiver {
			return int64(index)
		}
	}

	return -1
}

// get given route position in config file
func getRoutePosition(routes []*config.Route, receiver string, match map[string]string) int64 {
	for index, value := range routes {
		if value.Receiver == receiver {
			if match != nil {
				if value.Match["alertgroup"] == match["alertgroup"] {
					return int64(index)
				}
			} else {
				return int64(index)
			}
		}
	}

	return -1
}

func addRoute(cfg *config.Config, receiver int64, alert string) error {
	r := strconv.FormatInt(receiver, 10)
	match := make(map[string]string)
	match["alertgroup"] = alert

	p := getRoutePosition(cfg.Route.Routes, r, match)
	if p != -1 {
		log.Errorf("route already exists")
		return nil
	}

	route := &config.Route{
		Receiver: r,
		Continue: true,
		Match:    match,
	}

	cfg.Route.Routes = append(cfg.Route.Routes, route)

	return nil
}

func delRoute(cfg *config.Config, receiver int64, alert string) error {
	r := strconv.FormatInt(receiver, 10)
	match := make(map[string]string)
	match["alertgroup"] = alert

	p := getRoutePosition(cfg.Route.Routes, r, match)
	if p == -1 {
		return errors.New("given receiver doesn't have routes now")
	}

	cfg.Route.Routes[p] = cfg.Route.Routes[len(cfg.Route.Routes)-1]
	cfg.Route.Routes = cfg.Route.Routes[:len(cfg.Route.Routes)-1]

	return nil
}

func delAllRoutes(cfg *config.Config, receiver int64) error {
	r := strconv.FormatInt(receiver, 10)

	var routes []*config.Route
	for _, route := range cfg.Route.Routes {
		if r != route.Receiver {
			routes = append(routes, route)
		}
	}

	cfg.Route.Routes = routes

	return nil
}

func getAlertRuleGroupPages(text string) (*paginator.Paginator, error) {
	rg, err := getVMRuleGroups()
	if err != nil {
		return nil, err
	}

	var kb [][]tgbotapi.InlineKeyboardButton

	for _, name := range rg {
		kb = append(
			kb,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(name, fmt.Sprintf("%s:%s", text, name)),
			),
		)
	}

	rgp := paginator.New(adapter.NewSliceAdapter(kb), 10)

	return &rgp, nil
}

func getActiveSubscribePages(receiver int64, text string) (*paginator.Paginator, error) {
	cfg, err := getAlertmanagerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get alertmanager config: %s", err)
	}

	var kb [][]tgbotapi.InlineKeyboardButton
	r := strconv.FormatInt(receiver, 10)
	for _, value := range cfg.Route.Routes {
		if value.Receiver == r {
			name := value.Match["alertgroup"]
			kb = append(
				kb,
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(name, fmt.Sprintf("%s:%s", text, name)),
				),
			)
		}
	}

	if len(kb) == 0 {
		return nil, errors.New("routes with this receiver not found")
	}

	sp := paginator.New(adapter.NewSliceAdapter(kb), 10)

	return &sp, nil
}

func getVMRuleGroups() ([]string, error) {
	var rg []string

	namespace := viper.GetString("kube.namespace")

	rules, err := kubeCli.VMV1Beta1().VMRules(namespace).List(metav1.ListOptions{})
	for _, rule := range rules.Items {
		for _, group := range rule.Spec.Groups {
			rg = append(rg, group.Name)
		}
	}

	return rg, err
}

func getAlertmanagerConfig() (cfg *config.Config, err error) {
	namespace := viper.GetString("kube.namespace")
	name := viper.GetString("alertmanager.secret-name")

	s, err := kubeCli.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get alertmanager secret with config: %s", err)
	}

	if _, ok := s.Data["alertmanager.yaml"]; !ok {
		return nil, fmt.Errorf("secret not contain alertmanager.yaml file")
	}

	cfg, err = config.Load(string(s.Data["alertmanager.yaml"]))
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal alertmanager.yaml file: %s", err)
	}

	return cfg, nil
}

func writeAlertmanagerConfig(cfg *config.Config) error {
	namespace := viper.GetString("kube.namespace")
	name := viper.GetString("alertmanager.secret-name")

	s, err := kubeCli.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get alertmanager secret with config: %s", err)
	}

	if _, ok := s.Data["alertmanager.yaml"]; !ok {
		return fmt.Errorf("secret not contain alertmanager.yaml file")
	}

	data := cfg.String()
	s.Data["alertmanager.yaml"] = []byte(data)

	_, err = kubeCli.CoreV1().Secrets(namespace).Update(context.Background(), s, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update alertmanager secret with config: %s", err)
	}

	_, err = alertmanager.Reload(viper.GetString("alertmanager.url"))
	if err != nil {
		log.Errorf("failed reload alertmanager instance: %s", err)
	}

	return nil
}

// Truncate very big message
func truncateMessage(str string) string {
	truncateMsg := str
	if len(str) > 4095 { // telegram API can only support 4096 bytes per message
		log.Warn("msg", "Message is bigger than 4095, truncate...")
		// find the end of last alert, we do not want break the html tags
		i := strings.LastIndex(str[0:4080], "\n\n") // 4080 + "\n<b>[SNIP]</b>" == 4095
		if i > 1 {
			truncateMsg = str[0:i] + "\n<b>[SNIP]</b>"
		} else {
			truncateMsg = "Message is too long... can't send.."
			log.Warn("msg", "truncateMessage: Unable to find the end of last alert.")
		}
		return truncateMsg
	}
	return truncateMsg
}
