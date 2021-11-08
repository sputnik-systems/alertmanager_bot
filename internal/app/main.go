package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "alertmanager [subcommand]",
		Short: "alertmanager main command",
		RunE:  botRunE,
	}

	botRunCmd := &cobra.Command{
		Use:               "bot",
		Short:             "bot subcommand",
		RunE:              botRunE,
		PersistentPreRunE: botPreRunE,
	}

	rootCmd.PersistentFlags().String("log.level", "info", "log level")
	err := viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log.level"))
	if err != nil {
		return fmt.Errorf("failed to bind flag: %s", err)
	}

	botRunCmd.PersistentFlags().String("kube.namespace", "default", "specify current k8s namespace")
	botRunCmd.PersistentFlags().String("alertmanager.url", "http://localhost:9093", "alertmanager endpoint url")
	botRunCmd.PersistentFlags().String("alertmanager.secret-name", "", "alertmanager secret name which used to stora config")
	botRunCmd.PersistentFlags().String("bot.token", "", "bot token string (required)")
	botRunCmd.PersistentFlags().String("bot.templates-path", "templates/default.tmpl", "bot message templates path")
	botRunCmd.PersistentFlags().String("bot.webhook-url", "http://bot:8000/webhook", "bot webhook url")
	botRunCmd.PersistentFlags().String("bot.public-url", "http://localhost:8000", "bot webserver public url")
	botRunCmd.PersistentFlags().String("oidc.issuer-url", "", "OpenID issuer url")
	botRunCmd.PersistentFlags().String("oidc.client-id", "", "OpenID auth client ID")
	botRunCmd.PersistentFlags().String("oidc.client-secret", "", "OpenID auth client secret")

	persistentRequiredFlags := []string{
		"bot.token",
		"alertmanager.secret-name",
	}
	for _, value := range persistentRequiredFlags {
		err = botRunCmd.MarkPersistentFlagRequired(value)
		if err != nil {
			return fmt.Errorf("failed to mark flag \"%s\" persistent: %s", value, err)
		}
	}

	bindFlags := []string{
		"kube.namespace",
		"alertmanager.url",
		"alertmanager.secret-name",
		"bot.token",
		"bot.templates-path",
		"bot.webhook-url",
		"bot.public-url",
		"oidc.issuer-url",
		"oidc.client-id",
		"oidc.client-secret",
	}
	for _, value := range bindFlags {
		err = viper.BindPFlag(value, botRunCmd.PersistentFlags().Lookup(value))
		if err != nil {
			return fmt.Errorf("failed to bind flag \"%s\": %s", value, err)
		}
	}

	rootCmd.AddCommand(botRunCmd)

	err = rootCmd.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute command: %s", err)
	}

	return nil
}
