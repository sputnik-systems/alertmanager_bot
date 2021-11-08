package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/types"
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
		var tmpl string
		switch {
		case viper.GetString("oidc.issuer-url") != "":
			tmpl = "%s/auth/oidc"
		default:
			tmpl = "%s/auth/simple"
		}

		bot.RegistrationURL = fmt.Sprintf(tmpl, viper.GetString("bot.public-url"))
	}

	// init OpenID auth backend
	if viper.GetString("oidc.issuer-url") != "" {
		provider, err := oidc.NewProvider(context.Background(), viper.GetString("oidc.issuer-url"))
		if err != nil {
			return fmt.Errorf("failed to initialize OIDC provider: %s", err)
		}

		oc = oauth2.Config{
			Endpoint:     provider.Endpoint(),
			ClientID:     viper.GetString("oidc.client-id"),
			ClientSecret: viper.GetString("oidc.client-secret"),
			RedirectURL:  fmt.Sprintf("%s/callback", bot.RegistrationURL),
			// Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}

	// init bot
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
		http.HandleFunc("/auth/simple", simpleRegistrationHandler)
		http.HandleFunc("/auth/oidc", oidcRedirectHandler)
		http.HandleFunc("/auth/oidc/callback", oidcCallbackHandler)

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

// oidc registration proccessors
func oidcRedirectHandler(w http.ResponseWriter, r *http.Request) {
	receiver := r.URL.Query().Get("receiver")
	http.Redirect(w, r, oc.AuthCodeURL(generateStateOauthCookie(w, receiver)), http.StatusFound)
}

func oidcCallbackHandler(w http.ResponseWriter, r *http.Request) {
	stateb64 := r.URL.Query().Get("state")
	stateb, err := base64.URLEncoding.DecodeString(stateb64)
	if err != nil {
		log.Printf("failed to decode oidc callback state: %s", err)

		return
	}

	receiver, err := strconv.ParseInt(string(stateb), 10, 64)
	if err != nil {
		log.Printf("failed to parse receiver id from state \"%s\": %s", string(stateb), err)

		return
	}

	if err := tb.RegisterReceiver(receiver); err != nil {
		log.Printf("failed to register receiver %d: %s", receiver, err)
	}
}

func generateStateOauthCookie(w http.ResponseWriter, receiver string) string {
	var expiration = time.Now().Add(time.Hour)

	state := base64.URLEncoding.EncodeToString([]byte(receiver))
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)

	return state
}

// simple registration processor
func simpleRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	if viper.GetString("oidc.issuer-url") != "" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Simple registration is not supported with another auth types"))

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
}
