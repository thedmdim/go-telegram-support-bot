package main

import "os"


var chatID string = os.Getenv("CHAT_ID")
var token string = os.Getenv("TOKEN")

func main() {
	bot := NewBot(chatID, token)
	bot.Start()
}