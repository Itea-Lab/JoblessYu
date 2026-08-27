package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"JoblessYu/internal/bot"
	"JoblessYu/internal/config"
	"JoblessYu/internal/job"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bwmarrin/discordgo"
)

var (
	disbot    *bot.Bot
	session   *discordgo.Session
	pubKey    ed25519.PublicKey
	initOnce  sync.Once
	initError error
)

func initBot() error {
	cfg := config.Load()
	if cfg.DiscordToken == "" {
		return fmt.Errorf("DISCORD_BOT_TOKEN is missing")
	}

	pubKeyHex := os.Getenv("DISCORD_PUBLIC_KEY")
	if pubKeyHex == "" {
		return fmt.Errorf("DISCORD_PUBLIC_KEY is missing")
	}

	rawPubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil || len(rawPubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid DISCORD_PUBLIC_KEY hex: %w", err)
	}
	pubKey = ed25519.PublicKey(rawPubKey)

	ctx := context.Background()
	repo, err := job.NewJobRepository(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to init repository: %w", err)
	}

	svc := job.NewJobService(repo)

	b, err := bot.NewBot(cfg, svc)
	if err != nil {
		return fmt.Errorf("failed to init bot: %w", err)
	}
	disbot = b

	s, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return fmt.Errorf("failed to init discordgo session: %w", err)
	}
	session = s

	return nil
}

func getHeader(headers map[string]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

func verifySignature(body, timestamp, sigHex string) bool {
	if sigHex == "" || timestamp == "" {
		return false
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	msg := []byte(timestamp + body)
	return ed25519.Verify(pubKey, msg, sig)
}

func handler(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	initOnce.Do(func() {
		initError = initBot()
	})
	if initError != nil {
		log.Printf("Initialization error: %v", initError)
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"error":"initialization failure"}`,
		}, nil
	}

	// 1. Validate cryptographic Ed25519 signature from Discord
	sig := getHeader(req.Headers, "x-signature-ed25519")
	timestamp := getHeader(req.Headers, "x-signature-timestamp")

	if !verifySignature(req.Body, timestamp, sig) {
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "invalid request signature",
		}, nil
	}

	// 2. Parse Discord interaction payload
	var interaction discordgo.Interaction
	if err := json.Unmarshal([]byte(req.Body), &interaction); err != nil {
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "bad request json",
		}, nil
	}

	// 3. Handle PING (Type 1) for Discord endpoint verification
	if interaction.Type == discordgo.InteractionPing {
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"type":1}`,
		}, nil
	}

	// 4. Delegate application command, message component, and modal interactions
	ic := &discordgo.InteractionCreate{Interaction: &interaction}
	disbot.HandleInteraction(session, ic)

	// Since discordgo methods (InteractionRespond) handle the response via Discord REST API,
	// returning 200 OK completes the Lambda execution cycle.
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"status":"ok"}`,
	}, nil
}

func main() {
	lambda.Start(handler)
}
