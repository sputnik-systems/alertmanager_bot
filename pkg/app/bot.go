package app

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/go-telegram-bot-api/telegram-bot-api"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vcraescu/go-paginator/v2"

	"github.com/sputnik-systems/alertmanager_bot/pkg/alertmanager"
	"github.com/sputnik-systems/alertmanager_bot/pkg/kubernetes"
)

const (
	PrevPage = "prev" // previous page data
	NextPage = "next" // next page data
)

var (
	kbPages map[int64]*paginator.Paginator
	kubeCli *kubernetes.Clientset

	// registration is using as minimal 2 steps
	// right now we will use simple token for auth
	// we need to store previous user message body when user is trying to send random string
	// we must compare it with given token only if previous message was "/start"
	chatPrevMessage map[int64]string

	subscribe *regexp.Regexp

	// supported ssince v5.0.1
	// https://github.com/go-telegram-bot-api/telegram-bot-api/pull/418
	// need approving
	//
	// botCommands []tgbotapi.BotCommand{
	// 	tgbotapi.BotCommand{"/subscribe", "Subscribe to alert group"},
	// 	tgbotapi.BotCommand{"/alerts", "List active alerts"},
	// }
)

type Bot struct {
	botAPI  *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel
	done    chan struct{}
}

func NewBot(token string) (*Bot, error) {
	var b *tgbotapi.BotAPI
	var err error

	b, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("bot api init failed: %s", err)
	}

	if viper.GetString("log.level") == "debug" {
		b.Debug = true
	}

	// err = b.SetMyCommands(botCommands)
	// if err != nil {
	// 	return nil, ftm.Errorf("failed set bot commands: %s", err)
	// }

	u, err := tgbotapi.NewUpdateWithFilter(
		0,
		tgbotapi.UpdateType_Message,
		tgbotapi.UpdateType_ChannelPost,
		tgbotapi.UpdateType_CallbackQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("failed init update config: %s", err)
	}

	u.Timeout = 60

	updates, err := b.GetUpdatesChan(u)
	if err != nil {
		return nil, fmt.Errorf("failed get updates: %s", err)
	}

	n := &Bot{
		botAPI:  b,
		updates: updates,
	}

	return n, nil
}

func botPreRunE(cmd *cobra.Command, args []string) error {
	var err error

	err = tgbotapi.SetLogger(log)
	if err != nil {
		log.Errorf("failed set logger for telegram lib: %s", err)
	}

	var config *rest.Config
	kubeconfig := viper.GetString("kube.config")
	if kubeconfig == "" {
		log.Printf("using in-cluster configuration")
		config, err = rest.InClusterConfig()
	} else {
		log.Printf("using configuration from '%s'", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return fmt.Errorf("failed kubeconfig init: %s", err)
	}

	kubeCli, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed default kube client init: %s", err)
	}

	subscribe, err = regexp.Compile(`^(?P<command>/(?:un)?subscribe):(?P<alertgroup>.+)$`)
	if err != nil {
		return fmt.Errorf("failed compile subscribe command regexp: %s", err)
	}

	kbPages = make(map[int64]*paginator.Paginator)
	chatPrevMessage = make(map[int64]string)

	return nil
}

func botRunE(cmd *cobra.Command, args []string) error {
	token := viper.GetString("bot.token")

	b, err := NewBot(token)
	if err != nil {
		return err
	}

	go b.UpdateHandler()
	go b.StartServer()

	select {
	case <-b.done:
		return fmt.Errorf("execution finished")
	}
}

func (b *Bot) StartServer() {
	var ch struct{}

	http.HandleFunc("/webhook", b.webhookHandler)
	http.HandleFunc("/health", healthChekHandler)
	log.Errorf("failed web server execution: %s", http.ListenAndServe(":8080", nil))

	b.done <- ch
}

func healthChekHandler(w http.ResponseWriter, r *http.Request) {
	return
}

func (b *Bot) webhookHandler(w http.ResponseWriter, r *http.Request) {
	alerts, receiver, err := alertmanager.GetWebhookData(r)
	if err != nil {
		log.Errorf("failed to get webhook: %s", err)
	}

	chatID, err := strconv.ParseInt(receiver, 10, 64)
	if err != nil {
		log.Errorf("failed convert webhook receiver to int64: %s", err)
	}

	text, err := alertmanager.GetMessageText(
		viper.GetString("bot.templates-path"),
		viper.GetString("alertmanager.url"),
		chatID, alerts,
	)
	if err != nil {
		log.Errorf("failed generate text from alert list: %s", err)
	}

	text = truncateMessage(text)

	if strings.ReplaceAll(text, "\n", "") == "" {
		log.Errorf("webhook body is empty")
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML

	b.botAPI.Send(msg)
}

func (b *Bot) UpdateHandler() {
	for update := range b.updates {
		message := getMessage(update)

		switch {
		case update.CallbackQuery != nil:
			err := b.callbackQueryHandler(update.CallbackQuery)
			if err != nil {
				log.Errorf("failed handle callback query: %s", err)
			}
		case message == nil:
			continue
		case message.Text == "/stop":
			// disable this receiver
			// can be realized over removing routes only
			// receiver should be kept
			err := disableReceiver(message.Chat.ID)
			if err != nil {
				log.Errorf("failed to response %s command: %s", message.Text, err)
			}
		// before handling this commands
		// we must check that user registered
		case message.Text == "/start" ||
			message.Text == "/subscribe" ||
			message.Text == "/unsubscribe" ||
			message.Text == "/alerts":
			err := b.proccessSecureCommands(message)
			if err != nil {
				log.Errorf("failed proccessing secure commands: %s", err)
			}
		default:
			log.Debugf("user give message token: %s", message.Text)
			if chatPrevMessage[message.Chat.ID] == "/start" &&
				message.Text == viper.GetString("user.registration-token") {
				err := registerReceiver(message.Chat.ID)
				if err != nil {
					log.Errorf("failed to response %s command: %s", message.Text, err)
				} else {
					b.botAPI.Send(
						tgbotapi.NewMessage(message.Chat.ID, "You have been successfully registered"),
					)
				}
			}

			chatPrevMessage[message.Chat.ID] = ""
		}
	}
}

func getMessage(update tgbotapi.Update) *tgbotapi.Message {
	var message *tgbotapi.Message
	if update.Message != nil {
		message = update.Message
	} else if update.ChannelPost != nil {
		message = update.ChannelPost
	}

	return message
}

func (b *Bot) proccessSecureCommands(message *tgbotapi.Message) error {
	chatID := message.Chat.ID

	exists, err := b.isReceiverExists(chatID)
	if err != nil {
		log.Errorf("failed to check receiver registration: %s", err)
	}

	if exists {
		switch message.Text {
		case "/start":
			b.botAPI.Send(tgbotapi.NewMessage(chatID, "You have already registered"))
		case "/subscribe", "/unsubscribe":
			err := b.getSubscriptionKB(message)
			if err != nil {
				return fmt.Errorf("failed to response %s command: %s", message.Text, err)
			}
		case "/alerts":
			err := b.getAlertMessage(chatID)
			if err != nil {
				return fmt.Errorf("failed to response /alerts command: %s\n", err)
			}
		}
	} else {
		switch message.Text {
		case "/start":
			chatPrevMessage[chatID] = message.Text
			b.botAPI.Send(
				tgbotapi.NewMessage(chatID, "Please give your token string in second message"),
			)
		case "/subscribe", "/unsubscribe", "/alerts":
			b.botAPI.Send(
				tgbotapi.NewMessage(chatID, "You should register to accessing this commands"),
			)
		}
	}

	return nil
}

func (b *Bot) getAlertMessage(chatID int64) error {
	// prepare params for alerts request
	params := make(map[string]string)
	params["silenced"] = "false"
	params["inhibited"] = "false"
	params["unprocessed"] = "false"

	alerts, err := alertmanager.ListAlerts(
		viper.GetString("alertmanager.url"),
		strconv.FormatInt(chatID, 10),
		params,
	)
	if err != nil {
		return fmt.Errorf("failed to get alerts from alertmanager: %s", err)
	}

	text, err := alertmanager.GetMessageText(
		viper.GetString("bot.templates-path"),
		viper.GetString("alertmanager.url"),
		chatID, alerts,
	)
	if err != nil {
		return fmt.Errorf("failed generate text from alert list: %s", err)
	}

	text = truncateMessage(text)

	if strings.ReplaceAll(text, "\n", "") == "" {
		text = "no alerts"
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML

	b.botAPI.Send(msg)

	return nil
}

func (b *Bot) callbackQueryHandler(query *tgbotapi.CallbackQuery) error {
	var err error
	var msg tgbotapi.MessageConfig

	switch {
	case query.Data == NextPage || query.Data == PrevPage:
		msg, err = getKeyboardNextPage(query)
	case subscribe.Match([]byte(query.Data)):
		cfg, err := getAlertmanagerConfig()
		if err != nil {
			return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
		}

		r := subscribe.FindStringSubmatch(query.Data)
		if r[1] == "/subscribe" {
			err := addRoute(cfg, query.Message.Chat.ID, r[2])
			if err != nil {
				return fmt.Errorf("failed to add route: %s", err)
			}
		} else {
			err := delRoute(cfg, query.Message.Chat.ID, r[2])
			if err != nil {
				return fmt.Errorf("failed to del route: %s", err)
			}
		}

		err = writeAlertmanagerConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to save alertmanger config: %s", err)
		}
	}

	if err != nil {
		log.Errorf("failed generate next message: %s", err)
	}

	// delete old message
	b.botAPI.DeleteMessage(
		tgbotapi.DeleteMessageConfig{
			ChatID:    query.Message.Chat.ID,
			MessageID: query.Message.MessageID,
		},
	)

	// and send new one
	b.botAPI.Send(msg)

	return nil
}

func (b *Bot) getSubscriptionKB(message *tgbotapi.Message) error {
	var err error
	var msg tgbotapi.MessageConfig

	chatID := message.Chat.ID

	switch message.Text {
	case "/subscribe":
		msg = tgbotapi.NewMessage(chatID, "Available alert groups:")
		kbPages[chatID], err = getAlertRuleGroupPages(message.Text)
		if err != nil {
			return fmt.Errorf("failed get alert group keyborad buttons: %s", err)
		}
	case "/unsubscribe":
		msg = tgbotapi.NewMessage(chatID, "Active alert groups:")
		kbPages[chatID], err = getActiveSubscribePages(chatID, message.Text)
		if err != nil {
			if err.Error() == "routes with this receiver not found" {
				b.botAPI.Send(tgbotapi.NewMessage(chatID, "Active subscriptions not found"))
			}

			return fmt.Errorf("failed get active subscriptions keyboard: %s", err)
		}
	}

	kb, err := addPosButtons(chatID)
	if err != nil {
		return fmt.Errorf("keyborad button rows generation failed: %s", err)
	}

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		kb...,
	)

	b.botAPI.Send(msg)

	return nil
}

func (b *Bot) isReceiverExists(chatID int64) (bool, error) {
	cfg, err := getAlertmanagerConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get alertmanager url: %s", err)
	}

	r := strconv.FormatInt(chatID, 10)
	if pos := getReceiverPosition(cfg, r); pos == -1 {
		return false, nil
	}

	return true, nil
}

func registerReceiver(chatID int64) error {
	var err error

	cfg, err := getAlertmanagerConfig()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	err = addReceiverToConfig(cfg, chatID)
	if err != nil {
		return fmt.Errorf("failed to add receiver: %s", err)
	}

	err = writeAlertmanagerConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to save alertmanger config: %s", err)
	}

	return nil
}

func disableReceiver(chatID int64) error {
	var err error

	cfg, err := getAlertmanagerConfig()
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config from specified secret: %s", err)
	}

	err = delAllRoutes(cfg, chatID)
	if err != nil {
		return fmt.Errorf("failed deleting all routes for given receiver: %s", err)
	}

	err = writeAlertmanagerConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to save alertmanger config: %s", err)
	}

	return nil
}

func getKeyboardNextPage(query *tgbotapi.CallbackQuery) (msg tgbotapi.MessageConfig, err error) {
	var move int

	chatID := query.Message.Chat.ID

	switch query.Data {
	case NextPage:
		move, err = (*kbPages[chatID]).NextPage()
		if err != nil {
			return msg, err
		}
	case PrevPage:
		move, err = (*kbPages[chatID]).PrevPage()
		if err != nil {
			return msg, err
		}
	}

	(*kbPages[chatID]).SetPage(move)

	kb, err := addPosButtons(chatID)
	if err != nil {
		return msg, err
	}

	msg = tgbotapi.NewMessage(chatID, "Alert Groups:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		kb...,
	)

	return msg, nil
}

func addPosButtons(chatID int64) ([][]tgbotapi.InlineKeyboardButton, error) {
	kb := make([][]tgbotapi.InlineKeyboardButton, 0)
	(*kbPages[chatID]).Results(&kb)

	hasNext, err := (*kbPages[chatID]).HasNext()
	if err != nil {
		return nil, err
	}

	hasPrev, err := (*kbPages[chatID]).HasPrev()
	if err != nil {
		return nil, err
	}

	switch {
	case hasNext && hasPrev:
		kb = append(
			kb,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("< Prev", PrevPage),
				tgbotapi.NewInlineKeyboardButtonData("Next >", NextPage),
			),
		)
	case hasNext:
		kb = append(
			kb,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Next >", NextPage),
			),
		)
	case hasPrev:
		kb = append(
			kb, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("< Prev", PrevPage),
			),
		)
	}

	return kb, nil
}
