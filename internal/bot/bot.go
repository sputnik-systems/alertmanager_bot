package bot

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/common/model"
	"github.com/vcraescu/go-paginator/v2"
	"gopkg.in/tucnak/telebot.v3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sputnik-systems/alertmanager_bot/internal/alertmanager"
)

var (
	cmds = []telebot.Command{
		{Text: "/start", Description: "Register in alertmanager"},
		{Text: "/stop", Description: "Disable any alerting"},
		{Text: "/subscribe", Description: "Subscribe to alert group"},
		{Text: "/unsubscribe", Description: "Unsubscribe to alert group"},
		{Text: "/alerts", Description: "List active alerts"},
	}

	RegistrationURL = "http://example.org:8000/auth/simple"
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

	return nil
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

func (b *Bot) handleUnsubscribeCommand(m telebot.Context) error {
	receiver := m.Chat().ID
	if err := b.checkAuth(receiver); err != nil {
		return err
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
	defer m.Delete()

	callback := m.Callback()
	n := strings.Index(callback.Data, "|")
	if n < 0 {
		return fmt.Errorf("unexpected callback query: %s", callback.Data)
	}
	unique := callback.Data[1:n]
	data := callback.Data[n+len("|"):]

	switch unique {
	case "/page":
		b.switchPage(receiver, data)

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

		b.ac.Config.AddRoute(receiver, group)

		if _, err = b.ac.Reload(); err != nil {
			return fmt.Errorf("failed to reload alertmanager: %s", err)
		}
	case "/unsubscribe":
		group, err := b.findAlertGroupNameByPrefix(data)
		if err != nil {
			return fmt.Errorf("not found alert group by given prefix %s: %s", data, err)
		}

		b.ac.Config.RemoveRoute(receiver, group)

		if _, err = b.ac.Reload(); err != nil {
			return fmt.Errorf("failed to reload alertmanager: %s", err)
		}
	}

	return nil
}
