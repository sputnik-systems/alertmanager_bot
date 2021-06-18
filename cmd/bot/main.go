package main

import (
	"fmt"

	"github.com/sputnik-systems/alertmanager_bot/pkg/app"
)

func main() {
	err := app.Execute()
	if err != nil {
		fmt.Printf("failed to start bot: %s", err)
	}
}
