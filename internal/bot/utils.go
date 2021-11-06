package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	vm "github.com/VictoriaMetrics/operator/api/v1beta1"
	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
	"gopkg.in/tucnak/telebot.v3"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	ErrAuth = errors.New("authorization required")
)

func init() {
	// hack for support victorimaetrics custom resources with kube client
	vm.AddToScheme(scheme.Scheme)
}

func (b *Bot) createAlertRuleGroupPages(receiver int64) error {
	groups, err := b.getVMRuleGroups()
	if err != nil {
		return err
	}

	var buttons [][]telebot.InlineButton

	for _, name := range groups {
		buttons = append(
			buttons,
			[]telebot.InlineButton{
				{Unique: "/subscribe", Text: name, Data: name},
			},
		)
	}

	b.mux.Lock()
	defer b.mux.Unlock()

	pages := paginator.New(adapter.NewSliceAdapter(buttons), 10)
	b.pages[receiver] = &pages

	return nil
}

func (b *Bot) makeActiveSubscribePages(receiver int64) error {
	conf, err := b.ac.Config.Get()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config: %s", err)
	}

	var buttons [][]telebot.InlineButton
	r := strconv.FormatInt(receiver, 10)
	for _, value := range conf.Route.Routes {
		if value.Receiver == r {
			name := value.Match["alertgroup"]
			buttons = append(
				buttons,
				[]telebot.InlineButton{
					{Unique: "/unsubscribe", Text: name, Data: name},
				},
			)
		}
	}

	if len(buttons) == 0 {
		return fmt.Errorf("routes with receiver %s not found", r)
	}

	b.mux.Lock()
	defer b.mux.Unlock()

	pages := paginator.New(adapter.NewSliceAdapter(buttons), 10)
	b.pages[receiver] = &pages

	return nil
}

func (b *Bot) addPositionButtons(receiver int64) ([][]telebot.InlineButton, error) {
	buttons := make([][]telebot.InlineButton, 0)
	err := (*b.pages[receiver]).Results(&buttons)
	if err != nil {
		return nil, err
	}

	hasNext, err := (*b.pages[receiver]).HasNext()
	if err != nil {
		return nil, err
	}

	hasPrev, err := (*b.pages[receiver]).HasPrev()
	if err != nil {
		return nil, err
	}

	switch {
	case hasNext && hasPrev:
		buttons = append(
			buttons,
			[]telebot.InlineButton{
				{Unique: "/page", Text: "< Prev", Data: "prev"},
				{Unique: "/page", Text: "Next >", Data: "next"},
			},
		)
	case hasNext:
		buttons = append(
			buttons,
			[]telebot.InlineButton{
				{Unique: "/page", Text: "Next >", Data: "next"},
			},
		)
	case hasPrev:
		buttons = append(
			buttons,
			[]telebot.InlineButton{
				{Unique: "/page", Text: "< Prev", Data: "prev"},
			},
		)
	}

	return buttons, nil
}

func (b *Bot) switchPage(receiver int64, direction string) error {
	var move int
	var err error

	switch direction {
	case "next":
		move, err = (*b.pages[receiver]).NextPage()
		if err != nil {
			return err
		}
	case "prev":
		move, err = (*b.pages[receiver]).PrevPage()
		if err != nil {
			return err
		}
	}

	(*b.pages[receiver]).SetPage(move)

	return nil
}

func (b *Bot) getVMRuleGroups() ([]string, error) {
	rules := &vm.VMRuleList{}
	if err := b.kc.List(context.Background(), rules); err != nil {
		return nil, fmt.Errorf("failed to list VMRuleList: %s", err)
	}

	var groups []string
	for _, rule := range rules.Items {
		for _, group := range rule.Spec.Groups {
			groups = append(groups, group.Name)
		}
	}

	return groups, nil
}

func (b *Bot) findAlertGroupNameByPrefix(prefix string) (string, error) {
	groups, err := b.getVMRuleGroups()
	if err != nil {
		return "", err
	}

	for _, name := range groups {
		if strings.HasPrefix(name, prefix) {
			return name, nil
		}
	}

	return "", errors.New("no one alert group found")
}

func (b *Bot) checkAuth(receiver int64) error {
	if ok, err := b.ac.Config.IsReceiverExists(receiver); err != nil {
		return err
	} else if !ok {
		id := telebot.ChatID(receiver)
		if _, err := b.b.Send(id, "first you have to go auth flow"); err != nil {
			return err
		} else {
			return ErrAuth
		}
	}

	return nil
}

// Truncate very big message
func truncateMessage(str string) string {
	truncateMsg := str
	if len(str) > 4095 { // telegram API can only support 4096 bytes per message
		// log.Warn("msg", "Message is bigger than 4095, truncate...")

		// find the end of last alert, we do not want break the html tags
		i := strings.LastIndex(str[0:4080], "\n\n") // 4080 + "\n<b>[SNIP]</b>" == 4095
		if i > 1 {
			truncateMsg = str[0:i] + "\n<b>[SNIP]</b>"
		} else {
			truncateMsg = "Message is too long... can't send.."

			// log.Warn("msg", "truncateMessage: Unable to find the end of last alert.")
		}

		return truncateMsg
	}

	return truncateMsg
}
