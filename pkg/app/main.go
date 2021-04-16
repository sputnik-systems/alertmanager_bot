package app

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	log *logrus.Logger
)

func Execute() {
	var rootCmd = &cobra.Command{
		Use:               "alertmanager [subcommand]",
		Short:             "alertmanager main command",
		PersistentPreRunE: rootPreRunE,
	}

	var botRunCmd = &cobra.Command{
		Use:     "bot",
		Short:   "bot subcommand",
		RunE:    botRunE,
		PreRunE: botPreRunE,
	}

	rootCmd.PersistentFlags().String("log.level", "info", "log level")
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log.level"))

	botRunCmd.PersistentFlags().String("kube.config", "", "specify current k8s kubeconfig")
	botRunCmd.PersistentFlags().String("kube.namespace", "default", "specify current k8s namespace")
	botRunCmd.PersistentFlags().String("kube.selector", "", "specify k8s objects label selector")
	botRunCmd.PersistentFlags().String("alertmanager.url", "http://localhost:9093", "alertmanager endpoint url")
	botRunCmd.PersistentFlags().String("alertmanager.secret-name", "", "alertmanager secret name which used to stora config")
	botRunCmd.PersistentFlags().String("bot.token", "", "bot token string (required)")
	botRunCmd.PersistentFlags().String("bot.templates-path", "templates/default.tmpl", "bot message templates path")
	botRunCmd.PersistentFlags().String("bot.webhook-url", "http://bot:8080/webhook", "bot webhook url")
	botRunCmd.PersistentFlags().String("user.registration-token", "", "this token will be used when user try register")
	botRunCmd.MarkPersistentFlagRequired("bot.token")
	botRunCmd.MarkPersistentFlagRequired("alertmanager.secret-name")
	botRunCmd.MarkPersistentFlagRequired("user.registration-token")

	viper.BindPFlag("kube.config", botRunCmd.PersistentFlags().Lookup("kube.config"))
	viper.BindPFlag("kube.namespace", botRunCmd.PersistentFlags().Lookup("kube.namespace"))
	viper.BindPFlag("kube.selector", botRunCmd.PersistentFlags().Lookup("kube.selector"))
	viper.BindPFlag("alertmanager.url", botRunCmd.PersistentFlags().Lookup("alertmanager.url"))
	viper.BindPFlag("alertmanager.secret-name", botRunCmd.PersistentFlags().Lookup("alertmanager.secret-name"))
	viper.BindPFlag("bot.token", botRunCmd.PersistentFlags().Lookup("bot.token"))
	viper.BindPFlag("bot.templates-path", botRunCmd.PersistentFlags().Lookup("bot.templates-path"))
	viper.BindPFlag("bot.webhook-url", botRunCmd.PersistentFlags().Lookup("bot.webhook-url"))
	viper.BindPFlag("user.registration-token", botRunCmd.PersistentFlags().Lookup("user.registration-token"))

	rootCmd.AddCommand(botRunCmd)

	rootCmd.Execute()
}

func rootPreRunE(cmd *cobra.Command, args []string) error {
	log = logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	level, err := logrus.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		return fmt.Errorf("incorrect log level: %s", err)
	}

	log.SetLevel(level)

	return nil
}
