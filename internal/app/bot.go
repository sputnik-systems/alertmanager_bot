package app

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/sputnik-systems/alertmanager_bot/internal/alertmanager"
	"github.com/sputnik-systems/alertmanager_bot/internal/bot"
)

var (
	tb *bot.Bot
)

func botPreRunE(cmd *cobra.Command, args []string) error {
	var err error

	token := viper.GetString("bot.token")
	au := viper.GetString("alertmanager.url")
	wu := viper.GetString("bot.webhook-url")
	tp := viper.GetString("bot.templates-path")

	ac := types.NamespacedName{
		Namespace: viper.GetString("kube.namespace"),
		Name:      viper.GetString("alertmanager.secret-name"),
	}

	kc, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return fmt.Errorf("kube client initialization failed: %s", err)
	}

	tb, err = bot.New(token, au, wu, tp, ac, kc)
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

func healthChekHandler(w http.ResponseWriter, r *http.Request) {
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	alerts, receiver, err := alertmanager.GetWebhookData(r)
	if err != nil {
		log.Printf("failed to get webhook: %s", err)
	}

	if err = tb.ProcessWebhook(alerts, receiver); err != nil {
		log.Printf("failed to process webhook: %s", err)
	}
}
