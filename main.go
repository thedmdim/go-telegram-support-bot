package main

import (
	"log"
	"os"
	"strconv"
)

var supportChatID string = os.Getenv("GROUP")
var token string = os.Getenv("TOKEN")

func main() {
	i, err := strconv.ParseInt(supportChatID, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	bot := NewBot(i, token)
	bot.Start()
}
