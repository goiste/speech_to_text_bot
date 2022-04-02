package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"speech_to_text_bot/converter"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	errCantRecognize = "Не удалось распознать аудио."

	LogLevelDebug = "DEBUG"
	LogLevelError = "ERROR"
)

func (h *Handler) handleUpdate(upd tgbotapi.Update, errChan chan error) {
	if upd.ChannelPost != nil {
		h.handlePost(upd.ChannelPost, upd.ChannelPost.Chat.Title, true, errChan)
		return
	}

	if upd.Message != nil {
		message := upd.Message
		name := message.From.FirstName
		if message.ForwardFrom != nil {
			name = message.ForwardFrom.FirstName
		} else if message.ForwardFromChat != nil {
			name = message.ForwardFromChat.Title
		}

		h.handlePost(message, name, false, errChan)
		return
	}
}

func (h *Handler) handlePost(message *tgbotapi.Message, senderName string, voiceOnly bool, errChan chan error) {
	if message.Chat.ID != h.ownerID && !h.isTrustedID(message.Chat.ID) && !h.chatHasAccess(message.Chat) {
		fmt.Printf("unauthorized: id: %d; username: %s\n", message.Chat.ID, message.Chat.UserName)
		_ = h.SendLog(
			LogLevelError,
			fmt.Sprintf(
				"unauthorized\nid: %d\nusername: %s\nfirstname: %s\nlastname: %s\nmessage: %s",
				message.Chat.ID,
				message.Chat.UserName,
				message.Chat.FirstName,
				message.Chat.LastName,
				message.Text,
			),
		)
		_, err := h.sendText(message.Chat.ID, "access denied")
		if err != nil {
			errChan <- fmt.Errorf("send message error: %w\n", err)
		}
		return
	}

	if !h.needHandle(message, voiceOnly) {
		return
	}

	msgId, err := h.sendText(message.Chat.ID, "...")
	if err != nil {
		errChan <- fmt.Errorf("send message error: %w", err)
	}

	msgChan := make(chan string, 1)
	defer close(msgChan)

	msgCtx, msgCancel := context.WithCancel(context.Background())
	defer msgCancel()

	go h.messageUpdater(msgCtx, message.Chat.ID, msgId, msgChan)

	msgChan <- "Обработка..."

	fileData := []byte{}
	originalName := ""
	convertedName := ""
	sendConverted := false
	isVoice := false
	if message.Voice != nil {
		isVoice = true
		originalName = fmt.Sprintf("voice_message%d", time.Now().UnixMilli())
		convertedName, fileData, err = h.getFileData(message.Voice.FileID, "voice_message", message.Voice.MimeType, msgChan)
	} else if message.Audio != nil {
		originalName = message.Audio.FileName
		convertedName, fileData, err = h.getFileData(message.Audio.FileID, message.Audio.FileName, message.Audio.MimeType, msgChan)
	} else if message.Video != nil {
		sendConverted = true
		originalName = message.Video.FileName
		convertedName, fileData, err = h.getFileData(message.Video.FileID, message.Video.FileName, message.Video.MimeType, msgChan)
	} else if message.Document != nil {
		originalName = message.Document.FileName
		sendConverted = strings.Contains(message.Document.MimeType, "video")
		convertedName, fileData, err = h.getFileData(message.Document.FileID, message.Document.FileName, message.Document.MimeType, msgChan)
	}

	if err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(err.Error(), "convert error") {
			errMsg = "Не удалось конвертировать аудио."
		} else if strings.Contains(err.Error(), "too big") {
			errMsg = "Файл слишком большой (макс. 20 МБ)."
		}
		msgChan <- errMsg
		errChan <- fmt.Errorf("get file data error: %w", err)
		return
	}

	if sendConverted && len(fileData) > 0 {
		err = h.sendFile(message.Chat.ID, convertedName, "Конвертированное аудио", true, fileData)
		if err != nil {
			errChan <- fmt.Errorf("send converted error: %w", err)
		}
	}

	msgChan <- "Расшифровываем аудио..."
	recognized, recErr := h.ycl.Recognize(fmt.Sprintf("ch%dm%d", message.Chat.ID, message.MessageID), fileData)
	if recErr != nil {
		msgChan <- errCantRecognize
		errChan <- fmt.Errorf("recognition error: %w", recErr)
		return
	}

	result := ""
	if recognized == "" {
		result = errCantRecognize
	} else if len(recognized) > 1024 {
		textFileName := originalName + ".txt"
		err = h.sendFile(message.Chat.ID, textFileName, "", false, []byte(recognized))
		if err != nil {
			result = "Не удалось отправить файл с распознанным текстом."
			errChan <- fmt.Errorf("send text file error: %w", err)
		} else {
			result = fmt.Sprintf("Распознанный текст в файле %q", textFileName)
		}
	} else {
		prefix := "распознанный текст"
		if isVoice {
			prefix = fmt.Sprintf("%s сказал(а)", senderName)
		}
		result = fmt.Sprintf("<b>%s:</b>\n%s", prefix, recognized)
	}

	msgChan <- result

	if message.Chat.ID != h.ownerID {
		_ = h.SendLog(LogLevelDebug, fmt.Sprintf("%d ✓", message.Chat.ID))
	}
}

func (h *Handler) needHandle(message *tgbotapi.Message, voiceOnly bool) bool {
	if message.Voice == nil && message.Audio == nil && message.Video == nil && message.Document == nil {
		return false
	}

	if voiceOnly && message.Voice == nil {
		return false
	}

	if message.Document != nil && !strings.Contains(message.Document.MimeType, "audio") && !strings.Contains(message.Document.MimeType, "video") {
		return false
	}

	return true
}

func (h Handler) messageUpdater(ctx context.Context, chatID int64, messageID int, msgChan chan string) {
	seconds := 0
	currentMessage := ""
	ticker := time.NewTicker(time.Second)
	updateMessage := func() {
		if currentMessage != "" {
			_ = h.updateMessage(chatID, messageID, fmt.Sprintf("[%s] %s", formatSeconds(seconds), currentMessage))
		}
	}
	for {
		select {
		case <-ticker.C:
			seconds++
			updateMessage()
		case msg := <-msgChan:
			currentMessage = msg
			updateMessage()
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) updateMessage(chatID int64, messageID int, text string) error {
	_, err := h.bot.MakeRequest("editMessageText", tgbotapi.Params{
		"chat_id":    fmt.Sprintf("%d", chatID),
		"message_id": fmt.Sprintf("%d", messageID),
		"parse_mode": "html",
		"text":       text,
	})
	return err
}

func (h *Handler) sendText(id int64, message string) (int, error) {
	errMsg := tgbotapi.MessageConfig{
		BaseChat:  tgbotapi.BaseChat{ChatID: id},
		Text:      message,
		ParseMode: "html",
	}
	m, err := h.bot.Send(errMsg)
	if err != nil {
		return 0, err
	}
	return m.MessageID, err
}

func (h *Handler) sendFile(chatID int64, fileName, caption string, isAudio bool, data []byte) error {
	baseFileConfig := tgbotapi.BaseFile{
		BaseChat: tgbotapi.BaseChat{
			ChatID: chatID,
		},
		File: tgbotapi.FileBytes{
			Name:  fileName,
			Bytes: data,
		},
	}

	var err error

	if isAudio {
		_, err = h.bot.Send(tgbotapi.AudioConfig{
			BaseFile: baseFileConfig,
			Title:    fileName,
			Caption:  caption,
		})
	} else {
		_, err = h.bot.Send(tgbotapi.DocumentConfig{
			BaseFile: baseFileConfig,
			Caption:  caption,
		})
	}

	return err
}

func (h *Handler) SendLog(level, message string) error {
	_, err := h.sendText(h.ownerID, fmt.Sprintf("[%s]\n\n%s", level, message))
	return err
}

func (h *Handler) recognize(fileID string, fileName string) (string, error) {
	file, err := h.getMessageFile(fileID)
	if err != nil {
		return "", err
	}
	return h.ycl.Recognize(fileName, file)
}

func (h *Handler) getFileData(fileID, fileName, mimeType string, msgChan chan string) (convertedName string, fileData []byte, err error) {
	msgChan <- "Скачиваем файл..."
	if strings.Contains(mimeType, "ogg") {
		fileData, err = h.getMessageFile(fileID)
		return
	} else {
		content, contentError := h.getMessageFile(fileID)
		if contentError != nil {
			err = fmt.Errorf("get file content error: %w", contentError)
			return
		}
		msgChan <- "Конвертируем..."
		convertedName, fileData, err = converter.Convert(fileName, content)
		if err != nil {
			err = fmt.Errorf("convert error: %w", err)
			return
		}
	}
	return
}

func (h *Handler) getMessageFile(fileID string) ([]byte, error) {
	fileUrl, err := h.bot.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("get file url error: %w\n", err)
	}
	return getFile(fileUrl)
}

func (h *Handler) isTrustedID(id int64) bool {
	for _, tID := range h.trustedIDs {
		if id == tID {
			return true
		}
	}
	return false
}

func (h *Handler) chatHasAccess(chat *tgbotapi.Chat) bool {
	if chat.IsPrivate() {
		return false
	}

	admins, err := h.bot.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{ChatConfig: chat.ChatConfig()})
	if err != nil {
		fmt.Printf("get admins error: %v\n", err)
		return false
	}

	hasAccess := false
	for _, admin := range admins {
		if admin.User.ID == h.ownerID {
			hasAccess = true
			break
		}
	}

	return hasAccess
}
