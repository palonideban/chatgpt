package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"time"
)

var botStorage storage.Storage

type ConversationEntry struct {
	Prompt   gpt.Message
	Response gpt.Message
}

func main() {
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot, err := telegram.NewBot(config.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}

	var commandMenu []telegram.Command
	for _, command := range config.CommandMenu {
		if _, ok := telegram.CommandDescriptions[telegram.Command(command)]; ok {
			commandMenu = append(commandMenu, telegram.Command(command))
		}
	}

	if len(commandMenu) > 0 {
		_ = bot.SetCommandList(commandMenu...)
	} else {
		_ = bot.SetCommandList(telegram.DefaultCommandList...)
	}

	gptClient := &gpt.GPTClient{
		ApiKey: config.GPTToken,
	}

	// buffer up to 100 update messages
	updateChan := make(chan telegram.Update, 100)

	// create a pool of worker goroutines
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		go worker(updateChan, bot, gptClient, config)
	}

	// Here we can choose any type of implemented storage
	botStorage, err = storage.NewFileStorage("data")
	if err != nil {
		log.Fatalf("Error creating storage: %v", err)
	}

	for update := range bot.GetUpdateChannel(config.TimeoutValue) {
		// Ignore any non-Message Updates
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		chat, ok := botStorage.Get(chatID)
		if !ok {
			chat = &storage.Chat{
				ChatID: update.Message.Chat.ID,
				Settings: storage.ChatSettings{
					Temperature:  0.8,
					Model:        "gpt-3.5-turbo",
					MaxMessages:  config.MaxMessages,
					UseMarkdown:  false,
					SystemPrompt: "Anda adalah bot chatgpt yang membantu berdasarkan model bahasa OpenAI GPT.Anda adalah asisten yang membantu yang selalu mencoba membantu dan menjawab dengan informasi yang relevan.",
				},
				History:          make([]*storage.ConversationEntry, 0),
				ImageGenNextTime: time.Now(),
			}
			_ = botStorage.Set(chatID, chat)
		}

		// If no authorized users are provided, make the bot public
		if len(config.AuthorizedUserIds) > 0 {
			if !util.IsIdInList(update.Message.From.ID, config.AuthorizedUserIds) {
				if update.Message.Chat.Type == "private" {
					bot.Reply(chat.ChatID, update.Message.MessageID, "Maaf, Anda tidak memiliki akses ke bot ini.", false)
					log.Printf("Upaya akses tidak sah oleh pengguna %d: %s %s (%s)", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)

					// Notify the admin
					if config.AdminId > 0 {
						adminMessage := fmt.Sprintf("Upaya akses tidak sah oleh pengguna %d: %s %s (%s)", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)
						bot.Message(adminMessage, config.AdminId, false)
					}
				}
				continue
			}
		}

		// Send the Update to the worker goroutines via the channel
		updateChan <- update
	}
}

func formatHistory(history []gpt.Message) []string {
	if len(history) == 0 {
		return []string{"Kisah percakapan kosong."}
	}

	var historyMessage string
	var historyMessages []string
	characterCount := 0

	for i, message := range history {
		formattedLine := fmt.Sprintf("%d. %s: %s\n", i+1, util.Title(message.Role), message.Content)
		lineLength := len(formattedLine)

		if characterCount+lineLength > 4096 {
			historyMessages = append(historyMessages, historyMessage)
			historyMessage = ""
			characterCount = 0
		}

		historyMessage += formattedLine
		characterCount += lineLength
	}

	if len(historyMessage) > 0 {
		historyMessages = append(historyMessages, historyMessage)
	}

	return historyMessages
}

func messagesFromHistory(storageHistory []*storage.ConversationEntry) []gpt.Message {
	var history []*ConversationEntry
	for _, entry := range storageHistory {
		prompt := entry.Prompt
		response := entry.Response

		history = append(history, &ConversationEntry{
			Prompt:   gpt.Message{Role: prompt.Role, Content: prompt.Content},
			Response: gpt.Message{Role: response.Role, Content: response.Content},
		})
	}

	var messages []gpt.Message
	for _, entry := range history {
		messages = append(messages, entry.Prompt)
		if entry.Response != (gpt.Message{}) {
			messages = append(messages, entry.Response)
		}
	}
	return messages
}
