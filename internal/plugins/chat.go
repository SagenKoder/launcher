package plugins

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/SagenKoder/launcher/internal/applications"
	"github.com/SagenKoder/launcher/internal/config"
)

func init() {
	Register(Info{
		ID:            "chat",
		Name:          "AI Chat",
		IconPath:      applications.DebugResolveIcon("dialog-information"),
		Intro:         "Ask the assistant anything. Responses stream in real time.",
		Hint:          "Ask the AI",
		CloseOnSubmit: false,
		OnInit: func() (string, error) {
			cfg, err := loadChatConfig()
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("_Model: %s_", cfg.Model), nil
		},
		OnSubmitStream: chatStream,
	})
}

type chatConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

var (
	chatCfgOnce sync.Once
	chatCfg     chatConfig
	chatCfgErr  error

	chatHistoryMu sync.Mutex
	chatHistory   []openAIMessage
)

const (
	defaultChatBaseURL = "https://api.openai.com"
	defaultChatModel   = "gpt-5-chat"
)

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func loadChatConfig() (chatConfig, error) {
	chatCfgOnce.Do(func() {
		cfg, err := config.Load()
		if err != nil {
			chatCfgErr = err
			return
		}
		apiKey := strings.TrimSpace(cfg.Chat.APIKey)
		if apiKey == "" {
			chatCfgErr = fmt.Errorf("chat.api_key not set in config %q", config.Path())
			return
		}
		baseURL := strings.TrimSpace(cfg.Chat.BaseURL)
		if baseURL == "" {
			baseURL = defaultChatBaseURL
		}
		model := strings.TrimSpace(cfg.Chat.Model)
		if model == "" {
			model = defaultChatModel
		}
		chatCfg = chatConfig{APIKey: apiKey, BaseURL: strings.TrimRight(baseURL, "/"), Model: model}
	})
	return chatCfg, chatCfgErr
}

func chatStream(ctx context.Context, input string, emit func(string, bool)) error {
	cfg, err := loadChatConfig()
	if err != nil {
		return err
	}

	history := snapshotHistory()
	messages := append(history, openAIMessage{Role: "user", Content: input})

	body, err := json.Marshal(struct {
		Model    string          `json:"model"`
		Messages []openAIMessage `json:"messages"`
		Stream   bool            `json:"stream"`
	}{
		Model:    cfg.Model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		return err
	}

	endpoint := cfg.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		if len(buf) == 0 {
			return fmt.Errorf("chat API status %d", resp.StatusCode)
		}
		return fmt.Errorf("chat API status %d: %s", resp.StatusCode, strings.TrimSpace(string(buf)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	assistant := strings.Builder{}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("parse stream chunk: %w", err)
		}
		for _, choice := range chunk.Choices {
			piece := choice.Delta.Content
			if piece == "" {
				continue
			}
			assistant.WriteString(piece)
			for _, r := range piece {
				emit(string(r), false)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return fmt.Errorf("read stream: %w", err)
	}

	answer := assistant.String()
	if answer != "" {
		appendHistory(openAIMessage{Role: "user", Content: input}, openAIMessage{Role: "assistant", Content: answer})
	}
	return nil
}

func snapshotHistory() []openAIMessage {
	chatHistoryMu.Lock()
	defer chatHistoryMu.Unlock()
	dup := make([]openAIMessage, len(chatHistory))
	copy(dup, chatHistory)
	return dup
}

func appendHistory(messages ...openAIMessage) {
	chatHistoryMu.Lock()
	chatHistory = append(chatHistory, messages...)
	chatHistoryMu.Unlock()
}
