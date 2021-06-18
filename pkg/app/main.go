package app

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var log *logrus.Logger

func Execute() error {
	rootCmd := &cobra.Command{
		Use:               "alertmanager [subcommand]",
		Short:             "alertmanager main command",
		PersistentPreRunE: rootPreRunE,
	}

	botRunCmd := &cobra.Command{
		Use:     "bot",
		Short:   "bot subcommand",
		RunE:    botRunE,
		PreRunE: botPreRunE,
	}

	rootCmd.PersistentFlags().String("log.level", "info", "log level")
	err := viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log.level"))
	if err != nil {
		return fmt.Errorf("failed to bind flag: %s", err)
	}

	botRunCmd.PersistentFlags().String("kube.config", "", "specify current k8s kubeconfig")
	botRunCmd.PersistentFlags().String("kube.namespace", "default", "specify current k8s namespace")
	botRunCmd.PersistentFlags().String("kube.selector", "", "specify k8s objects label selector")
	botRunCmd.PersistentFlags().String("alertmanager.url", "http://localhost:9093", "alertmanager endpoint url")
	botRunCmd.PersistentFlags().String("alertmanager.secret-name", "", "alertmanager secret name which used to stora config")
	botRunCmd.PersistentFlags().String("bot.token", "", "bot token string (required)")
	botRunCmd.PersistentFlags().String("bot.templates-path", "templates/default.tmpl", "bot message templates path")
	botRunCmd.PersistentFlags().String("bot.webhook-url", "http://bot:8080/webhook", "bot webhook url")
	botRunCmd.PersistentFlags().String("user.registration-token", "", "this token will be used when user try register")

	persistentRequiredFlags := []string{
		"bot.token",
		"alertmanager.secret-name",
		"user.registration-token",
	}
	for _, value := range persistentRequiredFlags {
		err = botRunCmd.MarkPersistentFlagRequired(value)
		if err != nil {
			return fmt.Errorf("failed to mark flag \"%s\" persistent: %s", value, err)
		}
	}

	bindFlags := []string{
		"kube.config",
		"kube.namespace",
		"kube.selector",
		"alertmanager.url",
		"alertmanager.secret-name",
		"bot.token",
		"bot.templates-path",
		"bot.webhook-url",
		"user.registration-token",
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
