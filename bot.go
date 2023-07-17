package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/NicoNex/echotron/v3"
	_ "github.com/mattn/go-sqlite3"
)

type Bot struct {
	SupportChat int64
	SupportChatForLinks string
	Token string
	mu sync.Mutex
	Storage *sql.DB
	API echotron.API
}

func NewBot(chat, token string) *Bot {

	db, err := sql.Open("sqlite3", "bot.db")

	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(CreateTableIfNotExists)
	if err != nil {
		log.Fatal(err)
	}

	chatID, err := strconv.ParseInt(chat, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	return &Bot{
		SupportChat: chatID,
		SupportChatForLinks: string([]rune(chat)[4:]),
		Token: token,
		Storage: db,
		API: echotron.NewAPI(token),
	}
}

func (b *Bot) Start() {
	defer b.Storage.Close()
	
	for u := range echotron.PollingUpdates(b.Token) {
		// process only messages

		if chatID := u.ChatID(); chatID == b.SupportChat {
			// handle updates from our group
			switch {
			case u.Message != nil:
				go b.OnChatMessage(u.Message)
			case u.EditedMessage != nil:
				go b.OnChatEditedMessage(u.EditedMessage)
			}
		} else if chatID > 0 {
			// handle incoming requests from users
			switch {
			case u.Message != nil:
				go b.OnUserMessage(u.Message)
			case u.EditedMessage != nil:
				go b.OnUserEditedMessage(u.EditedMessage)
			}
		}
	}
}

func (b *Bot) RememberUser(userID int64) error {
	stmt := "INSERT INTO users (id) VALUES (?);"
	b.mu.Lock()
	_, err := b.Storage.Exec(stmt, userID)
	b.mu.Unlock()
	return err
}

func (b *Bot) RememberMessage(userID int64, userMessageID, chatMessageID int) error {

	stmt := "INSERT INTO messages (user_id, user_chat_message_id, support_chat_message_id) VALUES (?, ?, ?);"
	b.mu.Lock()
	_, err := b.Storage.Exec(stmt, userID, userMessageID, chatMessageID)
	b.mu.Unlock()
	return err
}

func (b *Bot) ForgetUser(userID int64) {

	stmt := "DELETE FROM users WHERE id = ?"
	b.mu.Lock()
	_, err := b.Storage.Exec(stmt, userID)
	log.Println("Error deleting user:", err)
	b.mu.Unlock()
	log.Println("DELTE user:", err)
}


func (b *Bot) FindMessage(query string, messageID int) (userID int64, userMessageID int) {
	result, _ := b.Storage.Query(query, messageID)
	result.Next()
	result.Scan(&userID, &userMessageID)
	result.Close()
	return
}

func (b *Bot) FindAssignedTo(userID int64) (specialist string) {
	stmt := "SELECT assigned_to FROM users WHERE id = ?;"
	result, err := b.Storage.Query(stmt, userID)
	log.Println(err)
	result.Next()
	result.Scan(&specialist)
	result.Close()
	return
}

func (b *Bot) AssignToMe(useriD int64, username string) error {
	stmt := "UPDATE users SET assigned_to = ? WHERE id = ?;"
	b.mu.Lock()
	_, err := b.Storage.Exec(stmt, username, useriD)
	b.mu.Unlock()
	return err
}

func (b *Bot) OnChatMessage(message *echotron.Message) {

	log.Println("got message in support chat")

	if string([]rune(message.Text)[0]) == "/" {
		
		switch message.Text {
		case "/help":
			go b.API.SendMessage("Доступные комманды:\n/take - назначить на себя\n/close - завершить разговор", message.Chat.ID, nil)
		case "/take":
			if message.ReplyToMessage == nil {
				b.API.SendMessage("Не получится! Этой командой надо ответить на сообщение", message.Chat.ID, nil)
				return
			}
			if message.From.Username == "" || message.From.Username == "GroupAnonymousBot" {
				b.API.SendMessage("Не получится! У вас либо скрыт username, либо вы анонимный админ", message.Chat.ID, nil)
				return
			}
			userID, _ := b.FindMessage(BySupportChatMessagID, message.ReplyToMessage.ID)
			err := b.AssignToMe(userID, message.From.Username)
			log.Println("Error assigning to me:", err)
			b.API.SendMessage(fmt.Sprintf("@%s взял в работу", message.From.Username), message.Chat.ID, nil)
		case "/close":
			if message.ReplyToMessage == nil {
				b.API.SendMessage("Не получится! Этой командой надо ответить на сообщение", message.Chat.ID, nil)
				return
			}
			userID, _ := b.FindMessage(BySupportChatMessagID, message.ReplyToMessage.ID)
			b.ForgetUser(userID)
			b.API.SendMessage("Разговор завершён", message.Chat.ID, nil)
		default:
			b.API.SendMessage("Такой команды нету", message.From.ID, nil)
		}
		return
	}

	if message.ReplyToMessage == nil {
		return
	} else if message.ReplyToMessage.From.ID != b.SupportChat {
		return
	}

	userID, _ := b.FindMessage(BySupportChatMessagID, message.ReplyToMessage.ID)

	if userID == 0 {
		opts := &echotron.MessageOptions{
			ParseMode: "Markdown",
		}
		content := fmt.Sprintf("Отправитель [сообщения](t.me/c/%s/%d) не найден в БД", b.SupportChatForLinks, message.ReplyToMessage.ID)
		b.API.SendMessage(content, b.SupportChat, opts)
		return
	}

	res, _ := b.API.SendMessage(message.Text, userID, nil)
	
	b.RememberMessage(userID, res.Result.ID, message.ID)
}

func (b *Bot) OnChatEditedMessage(message *echotron.Message) {
	log.Println("edit message in support chat")

	userID, messageID := b.FindMessage(BySupportChatMessagID, message.ID)
	b.API.EditMessageText(message.Text, echotron.NewMessageID(userID, messageID), nil)
}

func (b *Bot) OnUserMessage(message *echotron.Message) {
	log.Println("got message from user")

	b.RememberUser(message.From.ID)

	if string([]rune(message.Text)[0]) == "/" {		
		switch message.Text {
		case "/start":
			b.API.SendMessage("Welcome to support ARK Sign! Please describe your project.", message.From.ID, nil)
		default:
			b.API.SendMessage("There is no such command", message.From.ID, nil)
		}
		return
	}

	assignedTo := b.FindAssignedTo(message.From.ID)
	opts := &echotron.MessageOptions{
		ParseMode: "Markdown",
		DisableWebPagePreview: true,
	}

	var header string
	if message.From.Username == "" {
		header = fmt.Sprintf("#%d", message.From.ID)
	} else {
		header = fmt.Sprintf("[#%d](t.me/%s)", message.From.ID, message.From.Username)
	}
	if assignedTo != "" {
		header += " @" + assignedTo
	}
	
	message.Text = header + "\n\n" + message.Text
	res, _ := b.API.SendMessage(message.Text, b.SupportChat, opts)
	b.RememberMessage(message.From.ID, message.ID, res.Result.ID)
}

func (b *Bot) OnUserEditedMessage(message *echotron.Message) {
	log.Println("user edited message")

	assignedTo := b.FindAssignedTo(message.From.ID)
	opts := &echotron.MessageTextOptions{
		ParseMode: "Markdown",
		DisableWebPagePreview: true,
	}

	var header string
	if message.From.Username == "" {
		header = fmt.Sprintf("#%d", message.From.ID)
	} else {
		header = fmt.Sprintf("[#%d](t.me/%s)", message.From.ID, message.From.Username)
	}
	if assignedTo != "" {
		header += " @" + assignedTo
	}
	
	message.Text = header + "\n\n" + message.Text
	_, chatMessageID := b.FindMessage(ByUserChatMessageID, message.ID)
	b.API.EditMessageText(message.Text, echotron.NewMessageID(b.SupportChat, chatMessageID), opts)
}