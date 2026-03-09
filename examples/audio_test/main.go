// Groq STT test: transcription and translation via gateway.
// Requires gateway with Groq provider key (valid API key with speech-to-text access).
// Env: AROUTER_BASE_URL, AROUTER_API_KEY, AROUTER_AUDIO_FILE (default /tmp/test_audio.mp3).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
)

const (
	defaultBaseURL = "http://localhost:19080"
	defaultAPIKey  = "lr_live_fdfb1fd8db9aaf5981ec033ef49a357d1d16f68fbcbc33ed"
	testAudioPath  = "/tmp/test_audio.mp3"
)

func main() {
	baseURL := envOr("AROUTER_BASE_URL", defaultBaseURL)
	apiKey := envOr("AROUTER_API_KEY", defaultAPIKey)
	audioPath := envOr("AROUTER_AUDIO_FILE", testAudioPath)

	client := arouter.NewClient(baseURL, apiKey, arouter.WithTimeout(90*time.Second))

	fmt.Println("=== Groq STT SDK Test (Go) ===")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Transcription
	fmt.Println("[1] Transcription (groq/whisper-large-v3)")
	transResp, err := client.CreateTranscription(ctx, &arouter.TranscriptionRequest{
		Model:    "groq/whisper-large-v3",
		FilePath: audioPath,
	})
	if err != nil {
		log.Fatalf("  FAIL: %v", err)
	}
	fmt.Printf("  OK: text=%q\n", transResp.Text)
	if transResp.Language != "" {
		fmt.Printf("  language=%s\n", transResp.Language)
	}
	fmt.Println()

	// Translation
	fmt.Println("[2] Translation (groq/whisper-large-v3)")
	translationResp, err := client.CreateTranslation(ctx, &arouter.TranslationRequest{
		Model:    "groq/whisper-large-v3",
		FilePath: audioPath,
	})
	if err != nil {
		log.Fatalf("  FAIL: %v", err)
	}
	fmt.Printf("  OK: text=%q\n", translationResp.Text)
	fmt.Println()

	fmt.Println("=== All tests complete ===")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
