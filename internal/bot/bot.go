package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/common/model"
	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
	"gopkg.in/tucnak/telebot.v3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sputnik-systems/alertmanager_bot/internal/alertmanager"
	"github.com/sputnik-systems/alertmanager_bot/internal/monitoring/rules"
	prom "github.com/sputnik-systems/alertmanager_bot/internal/monitoring/rules/prometheus"
	vm "github.com/sputnik-systems/alertmanager_bot/internal/monitoring/rules/victoriametrics"
)

const (
	CallbackLimit = 64
)

var (
	cmds = []telebot.Command{
		{Text: "/start", Description: "Register in alertmanager"},
		{Text: "/stop", Description: "Disable any alerting"},
		{Text: "/subscribe", Description: "Subscribe to some alert group"},
		{Text: "/subscribeall", Description: "Subscribe to all alert groups"},
		{Text: "/unsubscribe", Description: "Revoke subscribtion"},
		{Text: "/alerts", Description: "List active alerts"},
	}

	RegistrationURL      = "http://example.org:8000/auth/simple"
	AuthFlowTextTemplate = `First you have to go auth <a href="%s?receiver=%d">flow</a>.`

	ErrAuth     = errors.New("authorization required")
	ErrNotFound = errors.New("no one alert group found")
)

type Bot struct {
	b     *telebot.Bot
	pages map[int64]*paginator.Paginator
	mux   sync.Mutex
	kc    client.Client
	ac    *alertmanager.Alertmanager
}

func New(token, au, wu, tp string, ac types.NamespacedName, kc client.Client) (*Bot, error) {
	a, err := alertmanager.New(au, wu, tp, ac, kc)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize alertmanager client: %s", err)
	}

	tb, err := telebot.NewBot(telebot.Settings{
		Token:     token,
		Poller:    &telebot.LongPoller{Timeout: 10 * time.Second},
		ParseMode: telebot.ModeHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telegram bot client: %s", err)
	}

	b := &Bot{
		b:     tb,
		pages: make(map[int64]*paginator.Paginator),
		kc:    kc,
		ac:    a,
	}

	if err := tb.SetCommands(cmds); err != nil {
		return nil, fmt.Errorf("failed to set telegram bot commands: %s", err)
	}

	tb.Handle("/start", b.handleStartCommand)
	tb.Handle("/stop", b.handleStopCommand)
	tb.Handle("/subscribe", b.handleSubscribeCommand)
	tb.Handle("/subscribeall", b.handleSubscribeAllCommand)
	tb.Handle("/unsubscribe", b.handleUnsubscribeCommand)
	tb.Handle("/alerts", b.handleAlertsCommand)

	tb.Handle(telebot.OnCallback, b.handleCallback)

	return b, nil
}

func (b *Bot) Start() {
	b.b.Start()
}

func (b *Bot) ProcessWebhook(alerts []*model.Alert, receiver string) error {
	text, err := b.ac.GetMessageText(receiver, alerts)
	if err != nil {
		return fmt.Errorf("failed generating text from alert list: %s", err)
	}
	text = truncateMessage(text)
	if strings.ReplaceAll(text, "\n", "") == "" {
		text = "no alerts"
	}

	id, err := strconv.ParseInt(receiver, 10, 64)
	if err != nil {
		return fmt.Errorf("failed converting receiver string to int64")
	}

	_, err = b.b.Send(telebot.ChatID(id), text)

	return err
}

func (b *Bot) RegisterReceiver(receiver int64) error {
	if err := b.ac.Config.RegisterReceiver(receiver); err != nil {
		return err
	}

	if _, err := b.ac.Reload(); err != nil {
		return fmt.Errorf("failed to reload alertmanager: %s", err)
	}

	return nil
}

func (b *Bot) handleStartCommand(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return err
	}

	return m.Send("You are already logined")
}

func (b *Bot) handleStopCommand(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return m.Send("first you have to go auth flow")
	}

	if err := b.ac.Config.DisableReceiver(m.Message().Chat.ID); err != nil {
		return err
	}

	if _, err := b.ac.Reload(); err != nil {
		return fmt.Errorf("failed to reload alertmanager: %s", err)
	}

	return nil
}

func (b *Bot) handleSubscribeCommand(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return err
	}

	if ok, err := b.ac.Config.IsRouteExists(receiver, nil); ok {
		return m.Send("You are already subscribed for all alert groups. Unsubscribe first.")
	} else if err != nil {
		return fmt.Errorf("failed checking route existence: %s", err)
	}

	if err := b.createAlertRuleGroupPages(receiver); err != nil {
		return fmt.Errorf("failed to create alert rule groups pages: %s", err)
	}

	ikb, err := b.addPositionButtons(receiver)
	if err != nil {
		return fmt.Errorf("failed to create inline keyboard: %s", err)
	}

	if _, err = b.ac.Reload(); err != nil {
		return fmt.Errorf("failed to reload alertmanager: %s", err)
	}

	return m.Send("Available alert groups:", &telebot.ReplyMarkup{InlineKeyboard: ikb})
}

func (b *Bot) handleSubscribeAllCommand(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return err
	}

	if ok, err := b.ac.Config.IsRouteExists(receiver, nil); ok {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed checking route existence: %s", err)
	}

	if err := b.ac.Config.AddRoute(receiver, nil); err != nil {
		return fmt.Errorf("failed adding route for all alert groups: %s", err)
	}

	if _, err := b.ac.Reload(); err != nil {
		return fmt.Errorf("failed to reload alertmanager: %s", err)
	}

	return nil
}

func (b *Bot) handleUnsubscribeCommand(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return err
	}

	if ok, err := b.ac.Config.IsRouteExists(receiver, nil); ok {
		return b.ac.Config.RemoveRoute(receiver, nil)
	} else if err != nil {
		return fmt.Errorf("failed checking route existence: %s", err)
	}

	if err := b.makeActiveSubscribePages(receiver); err != nil {
		return fmt.Errorf("failed to create active subscribe pages: %s", err)
	}

	ikb, err := b.addPositionButtons(receiver)
	if err != nil {
		return fmt.Errorf("failed to create inline keyboard: %s", err)
	}

	if _, err = b.ac.Reload(); err != nil {
		return fmt.Errorf("failed to reload alertmanager: %s", err)
	}

	return m.Send("Active alert groups:", &telebot.ReplyMarkup{InlineKeyboard: ikb})
}

func (b *Bot) handleAlertsCommand(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return err
	}

	// prepare params for alerts request
	params := make(map[string]string)
	params["silenced"] = "false"
	params["inhibited"] = "false"
	params["unprocessed"] = "false"
	alerts, err := b.ac.ListAlerts(strconv.FormatInt(receiver, 10), params)
	if err != nil {
		return fmt.Errorf("failed to get alerts from alertmanager: %s", err)
	}

	text, err := b.ac.GetMessageText(strconv.FormatInt(receiver, 10), alerts)
	if err != nil {
		return fmt.Errorf("failed generate text from alert list: %s", err)
	}
	text = truncateMessage(text)
	if strings.ReplaceAll(text, "\n", "") == "" {
		text = "no alerts"
	}

	return m.Send(text)
}

func (b *Bot) handleCallback(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return err
	}

	// LOG IT?!!
	// defer func() {
	// 	if err := m.Delete(); err != nil {
	// 		return fmt.Errorf("failed to delete old message: %s", err)
	// 	}
	// }
	defer func() {
		if err := m.Delete(); err != nil {
			log.Printf("failed to delete callback message: %s", err)
		}
	}()

	callback := m.Callback()
	n := strings.Index(callback.Data, "|")
	if n < 0 {
		return fmt.Errorf("unexpected callback query: %s", callback.Data)
	}
	unique := callback.Data[1:n]
	data := callback.Data[n+len("|"):]

	switch unique {
	case "/page":
		if err := b.switchPage(receiver, data); err != nil {
			return fmt.Errorf("failed to change keyboard page: %s", err)
		}

		ikb, err := b.addPositionButtons(receiver)
		if err != nil {
			return fmt.Errorf("failed to create inline keyboard: %s", err)
		}

		return m.Send("Available alert groups:", &telebot.ReplyMarkup{InlineKeyboard: ikb})
	case "/subscribe":
		group, err := b.findAlertGroupNameByPrefix(data)
		if err != nil {
			return fmt.Errorf("not found alert group by given prefix %s: %s", data, err)
		}

		match := make(map[string]string)
		match["alertgroup"] = group
		if err := b.ac.Config.AddRoute(receiver, match); err != nil {
			return err
		}

		if _, err = b.ac.Reload(); err != nil {
			return fmt.Errorf("failed to reload alertmanager: %s", err)
		}
	case "/unsubscribe":
		match, err := b.ac.Config.FindMatchByPrefix(receiver, data)
		if err != nil {
			return fmt.Errorf("failed to get match for given alert group prefix: %s", err)
		}

		if err := b.ac.Config.RemoveRoute(receiver, match); err != nil {
			return err
		}

		if _, err = b.ac.Reload(); err != nil {
			return fmt.Errorf("failed to reload alertmanager: %s", err)
		}
	}

	return nil
}

func (b *Bot) createAlertRuleGroupPages(receiver int64) error {
	groups, err := b.getRuleGroupNames()
	if err != nil {
		return err
	}

	var buttons [][]telebot.InlineButton
	length := CallbackLimit - len("\f/subscribe")
	for _, name := range groups {
		data := name
		if len(data) >= length {
			data = data[:length-1]
		}

		buttons = append(
			buttons,
			[]telebot.InlineButton{
				{Unique: "/subscribe", Text: name, Data: data},
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
	length := CallbackLimit - len("\f/unsubscribe")
	for _, value := range conf.Route.Routes {
		if value.Receiver == r {
			name := value.Match["alertgroup"]
			data := name
			if len(data) >= length {
				data = data[:length-1]
			}

			buttons = append(
				buttons,
				[]telebot.InlineButton{
					{Unique: "/unsubscribe", Text: name, Data: data},
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

func (b *Bot) getRuleGroupNames() ([]string, error) {
	var r []rules.Rule
	r = vm.Rules(b.kc)
	r = append(r, prom.Rules(b.kc)...)

	keys := make(map[string]struct{})
	groups := make([]string, 0)
	for _, rule := range r {
		for _, group := range rule.GetGroupNames() {
			if _, ok := keys[group]; !ok {
				keys[group] = struct{}{}
				groups = append(groups, group)
			}
		}
	}

	return groups, nil
}

func (b *Bot) findAlertGroupNameByPrefix(prefix string) (string, error) {
	groups, err := b.getRuleGroupNames()
	if err != nil {
		return "", err
	}

	for _, name := range groups {
		if strings.HasPrefix(name, prefix) {
			return name, nil
		}
	}

	return "", ErrNotFound
}

func (b *Bot) checkAuth(receiver int64) error {
	if ok, err := b.ac.Config.IsReceiverExists(receiver); err != nil {
		return err
	} else if !ok {
		id := telebot.ChatID(receiver)
		if _, err := b.b.Send(id, fmt.Sprintf(AuthFlowTextTemplate, RegistrationURL, receiver)); err != nil {
			return err
		} else {
			return ErrAuth
		}
	}

	return nil
}
