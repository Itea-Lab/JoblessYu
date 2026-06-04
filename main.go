// ==========================================
// SECTION 1: IMPORTS
// ==========================================
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// ==========================================
	// SECTION 2: INITIALIZE ENV AND DISCORD API
	// ==========================================

	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Note: .env file not found, using system env variables")
	}

	token := "Bot " + os.Getenv("DISCORD_BOT_TOKEN")
	dbURL := os.Getenv("DATABASE_URL")

	// Create Discord API Session
	disbot, err := discordgo.New(token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// ==========================================
	// SECTION 3: SLASH PREFIX COMMANDS
	// ==========================================

	// Define the handler for when a user triggers a slash command interaction
	disbot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Ignore interactions that aren't slash commands
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		// Handle the "/jobs" command
		if i.ApplicationCommandData().Name == "jobs" {
			// Acknowledge the command immediately to avoid a 3-second timeout.
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				fmt.Println("Error deferring interaction:", err)
				return
			}

			// Pass off execution to our DB handler at the bottom of the script
			handleDbConnectAndReply(s, i, dbURL)
		}
	})

	// Set Discord Intents
	disbot.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	// Open Connection to Discord
	err = disbot.Open()
	if err != nil {
		fmt.Println("Error opening connection", err)
		return
	}
	defer disbot.Close()

	// Register the slash command configuration to Discord's server UI
	command := &discordgo.ApplicationCommand{
		Name:        "jobs",
		Description: "Fetch IT Support roles with specific criteria",
		Options: []*discordgo.ApplicationCommandOption{
			// 1. REQUIRED OPTION (Must go first!)
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "level",
				Description: "Experience level required",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Intern", Value: "Intern"},
					{Name: "Junior", Value: "Junior"},
					{Name: "Senior", Value: "Senior"},
				},
			},
			// 2. OPTIONAL OPTION (Job Type)
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Job employment type",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Full-Time", Value: "fulltime"},
					{Name: "Part-Time", Value: "parttime"},
				},
			},
			// 3. OPTIONAL OPTION (Location)
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "location",
				Description: "Job Location preference",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Ho Chi Minh", Value: "HCM"},
					{Name: "Hanoi", Value: "Hanoi"},
				},
			},
		},
	}

	fmt.Println("Registering slash command with options...")
	registeredCmd, err := disbot.ApplicationCommandCreate(disbot.State.User.ID, "", command)
	if err != nil {
		log.Panicf("Cannot create slash command: %v", err)
	}

	fmt.Println("JoblessYu Vessel is now running. Press CTRL-C to exit.")

	// Cleanup command on close
	defer func() {
		fmt.Println("\nRemoving slash command...")
		_ = disbot.ApplicationCommandDelete(disbot.State.User.ID, "", registeredCmd.ID)
	}()

	// Keep running until Ctrl + C signal is received
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down")
}

// ==========================================
// SECTION 4: DB CONNECT AND REPLY (MVP Version)
// ==========================================

func handleDbConnectAndReply(s *discordgo.Session, i *discordgo.InteractionCreate, dbURL string) {
	ctx := context.Background()

	// Parse out options provided by the user
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	// Required option (e.g., "Intern", "Junior", "Senior")
	levelValue := optionMap["level"].StringValue()

	// Optional options
	typeValue := ""
	if opt, exists := optionMap["type"]; exists {
		typeValue = opt.StringValue()
	}

	locationValue := ""
	if opt, exists := optionMap["location"]; exists {
		locationValue = opt.StringValue()
	}

	// 1. Establish connection to Neon DB
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptr("❌ Failed to connect to Neon DB."),
		})
		fmt.Println("DB Connect Error:", err)
		return
	}
	defer conn.Close(ctx)

	// Update user with an initial status message
	statusMsg := fmt.Sprintf("Checking Neon DB for **%s** IT Support roles...", levelValue)
	if locationValue != "" {
		statusMsg += fmt.Sprintf(" in **%s**", locationValue)
	}
	if typeValue != "" {
		statusMsg += fmt.Sprintf(" (%s)", typeValue)
	}
	statusMsg += " =w="

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &statusMsg,
	})

	// 2. Build the dynamic SQL Query matching your exact Python MVP Schema
	baseQuery := "SELECT title, company, location, job_url FROM jobs WHERE title ILIKE $1"
	args := []interface{}{"%" + levelValue + "%"}
	placeholderCount := 2

	// Handle Location wildcard matching
	if locationValue != "" {
		if locationValue == "HCM" {
			baseQuery += fmt.Sprintf(" AND (location ILIKE $%d OR location ILIKE '%%Hồ Chí Minh%%')", placeholderCount)
			args = append(args, "%HCM%")
		} else {
			baseQuery += fmt.Sprintf(" AND location ILIKE $%d", placeholderCount)
			args = append(args, "%"+locationValue+"%")
		}
		placeholderCount++
	}

	// Handle Job Employment Type matching
	if typeValue != "" {
		var typeSearch string
		if typeValue == "fulltime" {
			typeSearch = "%full%"
		} else if typeValue == "parttime" {
			typeSearch = "%part%"
		}

		baseQuery += fmt.Sprintf(" AND job_type ILIKE $%d", placeholderCount)
		args = append(args, typeSearch)
		placeholderCount++
	}

	// Add sorting and cap it at 10 results
	baseQuery += fmt.Sprintf(" ORDER BY fetched_at DESC LIMIT 10")

	// 3. Query data
	rows, err := conn.Query(ctx, baseQuery, args...)
	if err != nil {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptr("❌ Error fetching filtered jobs from database."),
		})
		fmt.Println("Query Error:", err)
		return
	}
	defer rows.Close()

	var finalMessage string
	jobCount := 0

	// 4. Formulate the response layout
	for rows.Next() {
		var title, company, loc, url string
		err := rows.Scan(&title, &company, &loc, &url)
		if err != nil {
			continue
		}
		jobCount++
		finalMessage += fmt.Sprintf("📌 **%s**\n🏢 %s | 📍 %s\n🔗 <%s>\n\n", title, company, loc, url)
	}

	if jobCount == 0 {
		finalMessage = fmt.Sprintf("No recent jobs matching your criteria were found. (Filter: Level=%s, Loc=%s, Type=%s)", levelValue, locationValue, typeValue)
	}

	// 5. Edit the initial status message with our data payload
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &finalMessage,
	})
	if err != nil {
		fmt.Println("Error sending final interaction response:", err)
	}
}

// Simple helper utility to generate string pointers required by discordgo's WebhookEdit struct
func ptr(s string) *string {
	return &s
}
