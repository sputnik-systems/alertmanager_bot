package app

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/sputnik-systems/alertmanager_bot/internal/alertmanager"
	"github.com/sputnik-systems/alertmanager_bot/internal/bot"
)

var (
	tb *bot.Bot
	oc oauth2.Config
)

func botPreRunE(cmd *cobra.Command, args []string) error {
	var err error

	if viper.GetString("bot.public-url") != "" {
		bot.RegistrationURL = fmt.Sprintf("%s/auth", viper.GetString("bot.public-url"))
	}

	// init bot
	token := viper.GetString("bot.token")
	au := viper.GetString("alertmanager.url")
	wu := viper.GetString("bot.webhook-url")
	tp := viper.GetString("bot.templates-path")
	ns := viper.GetString("kube.namespace")
	acd := viper.GetString("alertmanager.dest-secret-name")
	acm := viper.GetString("alertmanager.manual-secret-name")

	kc, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return fmt.Errorf("kube client initialization failed: %s", err)
	}

	tb, err = bot.New(token, au, wu, tp, ns, acd, acm, kc)
	if err != nil {
		return fmt.Errorf("bot initialization failed: %s", err)
	}

	return nil
}

func botRunE(cmd *cobra.Command, args []string) error {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		http.HandleFunc("/health", healthChekHandler)
		http.HandleFunc("/webhook", webhookHandler)
		http.HandleFunc("/auth", registrationHandler)

		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Printf("web server execution failed: %s", err)

			wg.Done()
		}
	}()

	go func() {
		tb.Start()

		log.Printf("bot execution finished")

		wg.Done()
	}()

	wg.Wait()

	return nil
}

func healthChekHandler(w http.ResponseWriter, r *http.Request) {}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	alerts, receiver, err := alertmanager.GetWebhookData(r)
	if err != nil {
		log.Printf("failed to get webhook: %s", err)
	}

	if err = tb.ProcessWebhook(alerts, receiver); err != nil {
		log.Printf("failed to process webhook: %s", err)
	}
}

// simple registration processor
func registrationHandler(w http.ResponseWriter, r *http.Request) {
	if viper.GetString("oidc.issuer-url") != "" {
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte("Simple registration is not supported with another auth types")); err != nil {
			log.Printf("failed to write response body: %s", err)
		}

		return
	}

	receiver := r.URL.Query().Get("receiver")
	id, err := strconv.ParseInt(receiver, 10, 64)
	if err != nil {
		log.Printf("failed to parse receiver id \"%s\": %s", receiver, err)

		return
	}

	if err := tb.RegisterReceiver(id); err != nil {
		log.Printf("failed to register receiver %d: %s", id, err)
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Success")); err != nil {
		log.Printf("failed to write response body: %s", err)
	}
}
