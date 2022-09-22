package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	user_map := make(map[int64]string)
	file, err := os.Open("chains.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words := strings.Split(scanner.Text(), ";")
		if len(words) == 3 {
			i, err := strconv.ParseInt(words[0], 10, 64)
			if err != nil {
				panic(err)
			}
			user_map[i] = words[1] + " " + words[2]
		}
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_APITOKEN"))
	if err != nil {
		panic(err)
	}
	chat_id, err := strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)
	if err != nil {
		panic(err)
	}

	bot.Debug = true

	// Create a new UpdateConfig struct with an offset of 0. Offsets are used
	// to make sure Telegram knows we've handled previous values and we don't
	// need them repeated.
	updateConfig := tgbotapi.NewUpdate(0)

	// Tell Telegram we should wait up to 30 seconds on each request for an
	// update. This way we can get information just as quickly as making many
	// frequent requests without having to send nearly as many.
	updateConfig.Timeout = 30

	// Start polling Telegram for updates.
	updates := bot.GetUpdatesChan(updateConfig)

	// Let's go through each update that we're getting from Telegram.
	for update := range updates {
		// Telegram can send many types of updates depending on what your Bot
		// is up to. We only want to look at messages for now, so we can
		// discard any other updates.
		if update.Message == nil {
			continue
		} else if update.Message.Text == "/start" {
			if update.Message.Chat.ID == chat_id {
				result := user_map[update.Message.From.ID]
				if result == "" {
					result = chains_file(update.Message.From.ID, 0)
					user_map[update.Message.From.ID] = result
					result = "Вам назначено имя " + result
				} else {
					result = "Вам уже назначено имя " + result
				}
				msg := tgbotapi.NewMessage(update.Message.From.ID, result)
				bot.Send(msg)
			}
			continue
		}

		fake_user := user_map[update.Message.From.ID]
		if update.Message.Chat.ID != chat_id {
			if fake_user == "" {
				continue
			}
			msg := tgbotapi.NewMessage(chat_id, fake_user+":\n"+update.Message.Text)
			if _, err := bot.Send(msg); err != nil {
				panic(err)
			}
		}
		if fake_user == "" {
			fake_user = update.Message.From.UserName
		}

		for m_key, m_value := range user_map {
			_ = m_value
			if m_key != update.Message.From.ID {
				msg := tgbotapi.NewMessage(m_key, fake_user+":\n"+update.Message.Text)
				if _, err := bot.Send(msg); err != nil {
					panic(err)
				}
			} else if update.Message.Chat.ID == chat_id {
				msg := tgbotapi.NewMessage(m_key, "Вы написали:\n"+update.Message.Text)
				if _, err := bot.Send(msg); err != nil {
					panic(err)
				}
			}
		}

		// Now that we know we've gotten a new message, we can construct a
		// reply! We'll take the Chat ID and Text from the incoming message
		// and use it to create a new message.
		//		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		// We'll also say that this message is a reply to the previous message.
		// For any other specifications than Chat ID or Text, you'll need to
		// set fields on the `MessageConfig`.
		//		msg.ReplyToMessageID = update.Message.MessageID

		// Okay, we're sending our message off! We don't care about the message
		// we just sent, so we'll discard it.
		//		if _, err := bot.Send(msg); err != nil {
		// Note that panics are a bad way to handle errors. Telegram can
		// have service outages or network errors, you should retry sending
		// messages or more gracefully handle failures.
		//			panic(err)
		//		}
		//		msg1 := tgbotapi.NewMessage(chat_id, update.Message.Text)
		//		if _, err := bot.Send(msg1); err != nil {
		//			panic(err)
		//		}

	}
}

func chains_file(user_id int64, wmark int) string {
	var result, nflag, nname string = "", "", ""
	result = check_chains(strconv.FormatInt(user_id, 10), true)
	if wmark == 0 && result != "" {
		return "Вам уже назначено имя " + result
	}

	if wmark == 0 {
		i := 5
		for {
			nflag = generate_name("flags.txt")
			nname = generate_name("names.txt")
			if nname == "Error" || nflag == "Error" {
				if i == 0 {
					return "Error"
				} else {
					i -= 1
				}
			}
			result = check_chains(nflag+" "+nname, false)
			if result == "" {
				f, err := os.OpenFile("chains.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					panic(err)
				}
				defer f.Close()
				if _, err = f.WriteString(strconv.FormatInt(user_id, 10) + ";" + nflag + ";" + nname + "\n"); err != nil {
					panic(err)
				}
				return nflag + " " + nname
			}
		}
	}

	if wmark == 1 {
		return "Error"
	}
	return ""
}

func generate_name(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	lc, err := lineCounter(file)
	if err != nil {
		return "Error"
	}
	lc = rand.Intn(lc)

	file1, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file1.Close()
	scanner := bufio.NewScanner(file1)

	for scanner.Scan() {
		if lc == 0 {
			return scanner.Text()
		}
		lc -= 1
	}
	return "Error"
}

func check_chains(name string, wmark bool) string {
	file, err := os.Open("chains.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words := strings.Split(scanner.Text(), ";")
		if len(words) == 3 {
			if wmark {
				if name == words[0] {
					return words[1] + " " + words[2]
				}
			} else {
				if name == words[1]+" "+words[2] {
					return words[0]
				}
			}
		}
	}
	return ""
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
