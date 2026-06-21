package bot

import (
	"log"
	"sync"

	"JoblessYu/internal/config"
	"JoblessYu/internal/domain"
	"JoblessYu/internal/service"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session    *discordgo.Session
	cfg        *config.Config
	jobService *service.JobService

	jobsCache map[string][]domain.JobEntry
	cacheLock sync.RWMutex
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "jobs",
		Description: "Browse IT Support jobs with pagination",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "level",
				Description: "Job experience level",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Intern", Value: "Intern"},
					{Name: "Junior", Value: "Junior"},
					{Name: "Senior", Value: "Senior"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "time",
				Description: "Job type (optional)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Fulltime", Value: "Fulltime"},
					{Name: "Parttime", Value: "Parttime"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "location",
				Description: "Job location (optional)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "HCM (Ho Chi Minh)", Value: "HCM"},
					{Name: "HN (Ha Noi)", Value: "HN"},
				},
			},
		},
	},
}

func NewBot(cfg *config.Config, jobService *service.JobService) (*Bot, error) {
	token := "Bot " + cfg.DiscordToken
	session, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents =
		discordgo.IntentsGuildMessages |
			discordgo.IntentsDirectMessages |
			discordgo.IntentsGuilds |
			discordgo.IntentsMessageContent

	bot := &Bot{
		session:    session,
		cfg:        cfg,
		jobService: jobService,
		jobsCache:  make(map[string][]domain.JobEntry),
	}

	bot.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}
		if m.Content == "!testembed" {
			embed := &discordgo.MessageEmbed{
				Title:       "✅ Embed Test",
				Description: "If you can see this, embeds are working correctly.",
				Color:       0x57F287,
			}
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		}
	})

	bot.session.AddHandler(bot.handleInteraction)

	return bot, nil
}

func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return err
	}

	for _, cmd := range commands {
		existing, err := b.session.ApplicationCommands(b.session.State.User.ID, b.cfg.DiscordGuild)
		if err == nil {
			skip := false
			for _, existingCmd := range existing {
				if existingCmd.Name == cmd.Name {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}
		_, err = b.session.ApplicationCommandCreate(b.session.State.User.ID, b.cfg.DiscordGuild, cmd)
		if err != nil {
			log.Println("Failed to register slash command:", err)
		}
	}

	return nil
}

func (b *Bot) Stop() {
	b.session.Close()
}
