package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strconv"
	"time"
)

func commandRemoveUser(bot *telegram.Bot, update telegram.Update, chat *storage.Chat, config *Config) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to remove", false)
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()), false)
			return
		}

		newList := make([]int64, 0)
		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User will be removed: %d", userId), false)
			} else {
				newList = append(newList, auth)
			}
		}

		config.AuthorizedUserIds = newList
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, "Command successfully ended", false)
	}
}

func commandAddUser(bot *telegram.Bot, update telegram.Update, chat *storage.Chat, config *Config) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to add", false)
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()), false)
			return
		}

		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User already added: %d", userId), false)
				return
			}
		}

		config.AuthorizedUserIds = append(config.AuthorizedUserIds, userId)
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User successfully added: %d", userId), false)
	}
}

func commandReload(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	chatID := chat.ChatID
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)), false)
}

func commandTranslate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Harap berikan teks untuk diterjemahkan.Penggunaan: /tr <text>", false)
	} else {
		prompt := update.Message.CommandArguments()
		translationPrompt := fmt.Sprintf("Menerjemahkan teks berikut ke bahasa Inggris: \"%s\". Anda harus menjawab hanya dengan teks yang diterjemahkan tanpa penjelasan dan tanda kutip", prompt)
		systemPrompt := "Anda adalah asisten yang membantu yang diterjemahkan."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, translationPrompt)
	}
}

func commandGrammar(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Harap berikan teks untuk diperbaiki.Penggunaan: /gramar <text>", false)
	} else {
		prompt := update.Message.CommandArguments()
		grammarPrompt := fmt.Sprintf("Perbaiki teks berikut: \"%s\". Jawaban dengan teks yang dikoreksi saja.", prompt)
		systemPrompt := "Anda adalah asisten yang membantu yang mengoreksi tata bahasa."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, grammarPrompt)
	}
}

func commandEnhance(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Harap berikan teks untuk ditingkatkan.Penggunaan: /enhance <text>", false)
	} else {
		prompt := update.Message.CommandArguments()
		enhancePrompt := fmt.Sprintf("Tinjau dan tingkatkan teks berikut: \"%s\". Jawab dengan teks yang lebih baik.", prompt)
		systemPrompt := "Anda adalah asisten yang membantu yang mengulas teks untuk tata bahasa, gaya, dan hal -hal seperti itu."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, enhancePrompt)
	}
}

func commandHelp(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	helpText := `Daftar perintah yang tersedia dan deskripsinya:
/help - Menampilkan daftar perintah yang tersedia dan deskripsinya.
/start - Mengirimkan pesan yang ramah yang menggambarkan tujuan bot.
/history - Menunjukkan seluruh cerita yang dilestarikan pada saat percakapan dalam pemformatan yang indah.
/clear - Membersihkan riwayat percakapan untuk obrolan saat ini.
/rollback <n> - Menghapus pesan <n> terbaru dari sejarah percakapan untuk obrolan saat ini.
/tr <text> Menerjemahkan <text> Dalam bahasa apa pun dalam bahasa Inggris
/gramar <text> - Mengoreksi kesalahan tata bahasa <text>
/enhance <text> - Meningkatkan <text> с membantu GPT
/pap <text> - Menghasilkan gambar deskripsi <text> размера 512x512
/temperature <n> - Menetapkan suhu(Kreativitas) untuk GPT.Nilai yang diizinkan: 0,0 - 1.2`
	bot.Reply(chat.ChatID, update.Message.MessageID, helpText, false)
}

func commandHistory(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	historyMessages := formatHistory(messagesFromHistory(chat.History))
	for _, message := range historyMessages {
		bot.Reply(chat.ChatID, update.Message.MessageID, message, false)
	}
}

func commandStart(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	bot.Reply(chat.ChatID, update.Message.MessageID, "Halo! Saya seorang asisten turbo GPT-3.5, dan saya di sini untuk membantu Anda dengan pertanyaan atau tugas.Cukup tulis pertanyaan atau permintaan Anda, dan saya akan melakukan yang terbaik untuk membantu Anda!Untuk referensi, ketik /help.```Hehe```", true)
}

func commandClear(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	chat.History = nil
	bot.Reply(chat.ChatID, update.Message.MessageID, "Kisah percakapan telah dibersihkan.", false)
}

func commandRollback(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	number := 1
	if len(update.Message.CommandArguments()) > 0 {
		var err error
		number, err = strconv.Atoi(update.Message.CommandArguments())
		if err != nil || number < 1 {
			number = 1
		}
	}

	if number > len(chat.History) {
		number = len(chat.History)
	}

	if len(chat.History) > 0 {
		chat.History = chat.History[:len(chat.History)-number]
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Dihapus %d %s.", number, util.Pluralize(number, [3]string{"pesan", "pesan", "pesan"})), false)
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Kisah percakapan kosong.", false)
	}
}

func commandImagine(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat, config *Config) {
	now := time.Now()
	nextTime := chat.ImageGenNextTime
	if nextTime.After(now) && update.Message.From.ID != config.AdminId {
		nextTimeStr := nextTime.Format("15:04:05")
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Pembuatan gambar Anda berikutnya akan tersedia di %s.", nextTimeStr), false)
		return
	}

	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Harap berikan teks untuk menghasilkan gambar.Penggunaan: /pap <text>", false)
	} else {
		chat.ImageGenNextTime = now.Add(time.Second * 900)
		gptImage(bot, chat.ChatID, gptClient, update.Message.CommandArguments(), config)
	}
}

func commandTemperature(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Suhu saat ini %.1f.", chat.Settings.Temperature), false)
	} else {
		temperature, err := strconv.ParseFloat(update.Message.CommandArguments(), 64)
		if err != nil || temperature < 0.0 || temperature > 1.2 {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Nilai suhu yang salah.Itu harus dari 0,0 hingga 1,2.", false)
		} else {
			chat.Settings.Temperature = float32(temperature)
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Suhunya diatur %.1f.", temperature), false)
		}
	}
}
