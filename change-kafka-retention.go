package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

// LogEntry logs in json mod
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

func main() {
	logWithTime("Starting change retention...")

	kafkaIP := getEnv("KAFKA_IP", "i100ntesia-nma-kfk01.tc.egov.local")
	kafkaPort := getEnv("KAFKA_PORT", "9092")
	retentionMS := getEnv("RETENTION_MS", "1900000")              // 30m
	deleteRetentionMS := getEnv("DELETE_RETENTION_MS", "1900000") // 30m
	topicsEnv := getEnv("TOPICS", "")

	telegramToken := getEnv("TELEGRAM_TOKEN", "")
	telegramChatID := getEnv("TELEGRAM_CHAT_ID", "")

	config := sarama.NewConfig()
	config.Version = sarama.V3_1_0_0 // Kafka 3.1.0

	client, err := sarama.NewClient([]string{kafkaIP + ":" + kafkaPort}, config)
	if err != nil {
		message := fmt.Sprintf("Fail to create Kafka client: %v 💩", err)
		logWithTime(message)
		sendTelegramMessage(telegramToken, telegramChatID, "💩Error💩\n"+message)
		os.Exit(1)
	}
	defer client.Close()

	var topics []string
	if topicsEnv != "" {
		topicItems := strings.Split(topicsEnv, ",")
		for _, topic := range topicItems {
			topics = append(topics, strings.TrimSpace(topic))
		}
	} else {
		topics, err = client.Topics()
		if err != nil {
			message := fmt.Sprintf("Fail to get topics list: %v 💩", err)
			logWithTime(message)
			sendTelegramMessage(telegramToken, telegramChatID, "💩Error💩\n"+message)
			os.Exit(1)
		}
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		message := fmt.Sprintf("Fail to create Kafka admin-client: %v 💩", err)
		logWithTime(message)
		sendTelegramMessage(telegramToken, telegramChatID, "💩Error💩\n"+message)
		os.Exit(1)
	}
	defer admin.Close()

	for _, topic := range topics {
		logWithTime(fmt.Sprintf("Checking retention and delete.retention settings for topic %s 🚬", topic))

		resource := sarama.ConfigResource{
			Type: sarama.TopicResource,
			Name: topic,
		}

		configs, err := admin.DescribeConfig(resource)
		if err != nil {
			message := fmt.Sprintf("Fail to describe config for topic %s: %v 💩", topic, err)
			logWithTime(message)
			sendTelegramMessage(telegramToken, telegramChatID, "💩Error💩\n"+message)
			continue
		}

		currentRetentionMS := getConfigValue(configs, "retention.ms")
		currentDeleteRetentionMS := getConfigValue(configs, "delete.retention.ms")

		configToUpdate := make(map[string]*string)

		if currentRetentionMS != "" && compareMSValues(currentRetentionMS, retentionMS) {
			logWithTime(fmt.Sprintf("Need to update retention.ms: currentRetentionMS=%s > retentionMS=%s", currentRetentionMS, retentionMS))
			configToUpdate["retention.ms"] = &retentionMS
		} else {
			// Save current retention 
			configToUpdate["retention.ms"] = &currentRetentionMS
		}

		if currentDeleteRetentionMS != "" && compareMSValues(currentDeleteRetentionMS, deleteRetentionMS) {
			logWithTime(fmt.Sprintf("Need to update delete.retention.ms: currentDeleteRetentionMS=%s > deleteRetentionMS=%s", currentDeleteRetentionMS, deleteRetentionMS))
			configToUpdate["delete.retention.ms"] = &deleteRetentionMS
		} else {
			// Save current delete.retention
			configToUpdate["delete.retention.ms"] = &currentDeleteRetentionMS
		}

		if len(configToUpdate) > 0 {
			err = admin.AlterConfig(sarama.TopicResource, topic, configToUpdate, false)
			if err != nil {
				message := fmt.Sprintf("Fail to update configs for topic %s: %v 💩", topic, err)
				logWithTime(message)
				sendTelegramMessage(telegramToken, telegramChatID, "💩Error💩\n"+message)
			} else {
				logWithTime(fmt.Sprintf("Configs updated for topic %s 🥒: retention.ms=%s, delete.retention.ms=%s", topic, *configToUpdate["retention.ms"], *configToUpdate["delete.retention.ms"]))
			}
		} else {
			logWithTime(fmt.Sprintf("No updates needed for topic %s, current: retention.ms=%s, delete.retention.ms=%s", topic, currentRetentionMS, currentDeleteRetentionMS))
		}
	}

	logWithTime("🎊🎊🎊 Retention change process completed successfully. 🎊🎊🎊")
}

func logWithTime(message string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   message,
	}
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("Failed to marshal log entry to JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// getEnv take dafault value if env not exist
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	logWithTime(fmt.Sprintf("Couldn't find external var %s. Use %s by default.🦄", key, defaultValue))
	return defaultValue
}

// sendTelegramMessage sent message to tg
func sendTelegramMessage(token, chatID, message string) {
	if token == "" || chatID == "" {
		logWithTime("Telegram token or chat ID is not set, skipping Telegram notification.💩")
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		logWithTime(fmt.Sprintf("💩 Failed to send Telegram message: %v", err))
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			logWithTime(fmt.Sprintf("💩 Telegram API returned non-OK status: %v", resp.Status))
		} else {
			logWithTime("🤳 Error notification sent to Telegram successfully.")
		}
	}
}

// getConfigValue take current value from sarama.ConfigEntries
func getConfigValue(configEntries []sarama.ConfigEntry, configName string) string {
	for _, entry := range configEntries {
		if entry.Name == configName {
			return entry.Value
		}
	}
	return ""
}

// compareMSValues compare current and target value of values in ms
func compareMSValues(currentValue, targetValue string) bool {
	currentMS, err1 := strconv.ParseInt(currentValue, 10, 64)
	targetMS, err2 := strconv.ParseInt(targetValue, 10, 64)
	if err1 != nil || err2 != nil {
		logWithTime("Failed to parse retention values 💩.")
		return false
	}
	return currentMS > targetMS
}

