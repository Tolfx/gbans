package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
)

const (
	maxEmbedFields = 25
	// TODO ensure these
	//minUsernameChars    = 2
	//maxUsernameChars    = 32
	//maxAuthorChars      = 256
	maxFieldNameChars   = 256
	maxFieldValueChars  = 1024
	maxDescriptionChars = 2048
)

var (
	errCommandFailed = errors.New("Command failed")
	errTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

func (bot *Discord) SendEmbed(channelId string, message *discordgo.MessageEmbed) error {
	if bot.session == nil {
		return nil
	}
	if _, errSend := bot.session.ChannelMessageSendEmbed(channelId, message); errSend != nil {
		return errSend
	}
	return nil
}

// Discord implements the ChatBot interface for the discord chat platform.
type Discord struct {
	session         *discordgo.Session
	app             *App
	ctx             context.Context
	logger          *zap.Logger
	connectedMu     *sync.RWMutex
	commandHandlers map[botCmd]botCommandHandler
	Connected       atomic.Bool
	retryCount      int64
}

// NewDiscord instantiates a new, unconnected, discord instance
func NewDiscord(app *App) (*Discord, error) {
	bot := Discord{
		ctx:         app.ctx,
		app:         app,
		logger:      app.logger.Named("discord"),
		session:     nil,
		connectedMu: &sync.RWMutex{},
		Connected:   atomic.Bool{},
		retryCount:  -1,
	}
	bot.commandHandlers = map[botCmd]botCommandHandler{
		cmdBan:      bot.onBan,
		cmdCheck:    bot.onCheck,
		cmdCSay:     bot.onCSay,
		cmdFind:     bot.onFind,
		cmdKick:     bot.onKick,
		cmdMute:     bot.onMute,
		cmdPlayers:  bot.onPlayers,
		cmdPSay:     bot.onPSay,
		cmdSay:      bot.onSay,
		cmdServers:  bot.onServers,
		cmdUnban:    bot.onUnban,
		cmdSetSteam: bot.onSetSteam,
		cmdHistory:  bot.onHistory,
		cmdFilter:   bot.onFilter,
		cmdLog:      bot.onLog,
		//cmdStats:    bot.onStats,
	}
	return &bot, nil
}

func (bot *Discord) Start(ctx context.Context, token string) error {
	// Immediately connects, so we connect within the Start func
	session, errNewSession := discordgo.New("Bot " + token)
	if errNewSession != nil {
		return errors.Wrapf(errNewSession, "Failed to connect to discord. discord unavailable")
	}
	defer func() {
		if bot.session != nil {
			if errDisc := bot.session.Close(); errDisc != nil {
				bot.logger.Error("Failed to cleanly shutdown discord", zap.Error(errDisc))
			}
		}
	}()

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onConnect)
	session.AddHandler(bot.onDisconnect)
	session.AddHandler(bot.onInteractionCreate)

	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers
	//session.Identify.Intents |= discordgo.IntentGuildPresences

	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := session.Open(); errSessionOpen != nil {
		return errors.Wrap(errSessionOpen, "Error opening discord connection")
	}

	bot.session = session
	if errRegister := bot.botRegisterSlashCommands(); errRegister != nil {
		bot.logger.Error("Failed to register discord slash commands", zap.Error(errRegister))
	}
	<-ctx.Done()
	return nil
}

func (bot *Discord) onReady(s *discordgo.Session, _ *discordgo.Ready) {
	bot.Connected.Store(true)
	bot.logger.Info("Service state changed", zap.String("state", "ready"))
}

func (bot *Discord) onConnect(session *discordgo.Session, _ *discordgo.Connect) {
	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      config.General.ExternalUrl,
				State:    "state field",
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
		},
		AFK:    false,
		Status: "https://github.com/leighmacdonald/gbans",
	}
	if errUpdateStatus := session.UpdateStatusComplex(status); errUpdateStatus != nil {
		bot.logger.Error("Failed to update status complex", zap.Error(errUpdateStatus))
	}
	bot.logger.Info("Service state changed", zap.String("state", "connected"))
}

func (bot *Discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.Connected.Store(false)
	bot.retryCount++
	bot.logger.Info("Service state changed", zap.String("state", "disconnected"))
}

func (bot *Discord) sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
	if !bot.Connected.Load() {
		bot.logger.Error("Tried to send message to disconnected client")
		return nil
	}
	if wrap {
		msg = discordMsgWrapper + msg + discordMsgWrapper
	}
	if len(msg) > discordMaxMsgLen {
		return errTooLarge
	}
	_, errChannelMessageSend := session.ChannelMessageSend(channelId, msg)
	if errChannelMessageSend != nil {
		return errors.Wrapf(errChannelMessageSend, "Failed sending success (paged) response for interaction")
	}
	return nil
}

func (bot *Discord) sendInteractionMessageEdit(session *discordgo.Session, interaction *discordgo.Interaction, response botResponse) error {
	if !bot.Connected.Load() {
		bot.logger.Error("Tried to send message edit to disconnected client")
		return nil
	}
	edit := &discordgo.WebhookEdit{
		Embeds:          nil,
		AllowedMentions: nil,
	}
	var embeds []*discordgo.MessageEmbed
	switch response.MsgType {
	case mtString:
		val, ok := response.Value.(string)
		if ok && val != "" {
			edit.Content = &val
			if len(*edit.Content) > discordMaxMsgLen {
				return errTooLarge
			}
		}
	case mtEmbed:
		embeds = append(embeds, response.Value.(*discordgo.MessageEmbed))
		edit.Embeds = &embeds
	}
	_, errResp := session.InteractionResponseEdit(interaction, edit)
	return errResp
}

func (bot *Discord) Send(channelId string, message string, wrap bool) error {
	return bot.sendChannelMessage(bot.session, channelId, message, wrap)
}

func addFieldInline(embed *discordgo.MessageEmbed, logger *zap.Logger, title string, value string) {
	addFieldRaw(embed, logger, title, value, true)
}

func addField(embed *discordgo.MessageEmbed, logger *zap.Logger, title string, value string) {
	addFieldRaw(embed, logger, title, value, false)
}

func addFieldInt64Inline(embed *discordgo.MessageEmbed, logger *zap.Logger, title string, value int64) {
	addField(embed, logger, title, fmt.Sprintf("%d", value))
}

//func addFieldInt64(embed *discordgo.MessageEmbed, logger *zap.Logger, title string, value int64) {
//	addField(embed, logger, title, fmt.Sprintf("%d", value))
//}

//func addAuthor(embed *discordgo.MessageEmbed, person model.Person) {
//	name := person.PersonaName
//	if name == "" {
//		name = person.SteamID.String()
//	}
//	embed.Author = &discordgo.MessageEmbedAuthor{URL: person.ToURL(), Name: name}
//}

func addAuthorProfile(embed *discordgo.MessageEmbed, logger *zap.Logger, person model.UserProfile) {
	name := person.Name
	if name == "" {
		name = person.SteamID.String()
	}
	if name == "" {
		logger.Warn("Value cannot be empty, dropping field", zap.String("field", "name"))
		return
	}
	embed.Author = &discordgo.MessageEmbedAuthor{URL: person.ToURL(), Name: name}
}

func addLink(embed *discordgo.MessageEmbed, logger *zap.Logger, value model.Linkable) {
	url := value.ToURL()
	if len(url) > 0 {
		addFieldRaw(embed, logger, "Link", url, false)
	}
}

func addFieldRaw(embed *discordgo.MessageEmbed, logger *zap.Logger, title string, value string, inline bool) {
	if len(embed.Fields) >= maxEmbedFields {
		logger.Warn("Dropping embed fields. Already at max count", zap.Int("max", maxEmbedFields))
		return
	}
	if len(title) == 0 {
		logger.Warn("Title cannot be empty, dropping field")
		return
	}
	if len(value) == 0 {
		logger.Warn("Value cannot be empty, dropping field", zap.String("field", title))
		return
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   truncate(title, maxFieldNameChars),
		Value:  truncate(value, maxFieldValueChars),
		Inline: inline,
	})
}

func truncate(str string, maxLen int) string {
	if len(str) > maxLen {
		return str[:maxLen]
	}
	return str
}

func addFieldsSteamID(embed *discordgo.MessageEmbed, logger *zap.Logger, steamId steamid.SID64) {
	addFieldInline(embed, logger, "STEAM", string(steamid.SID64ToSID(steamId)))
	addFieldInline(embed, logger, "STEAM3", string(steamid.SID64ToSID3(steamId)))
	addFieldInline(embed, logger, "SID64", steamId.String())
}

// ChatBot defines a interface for communication with 3rd party service bots
// Currently this is only used for discord, but other providers such as
// Guilded, Matrix, IRC, etc. are planned.
// TODO decouple embed's from discordgo
type ChatBot interface {
	Start(ctx context.Context, token string, eventChan chan model.ServerEvent) error
	Send(channelId string, message string, wrap bool) error
	SendEmbed(channelId string, message *discordgo.MessageEmbed) error
}

// LevelColors is a struct of the possible colors used in Discord color format (0x[RGB] converted to int)
type LevelColors struct {
	Debug int
	Info  int
	Warn  int
	Error int
	Fatal int
}

// DefaultLevelColors is a struct of the default colors used
var DefaultLevelColors = LevelColors{
	Debug: 10170623,
	Info:  3581519,
	Warn:  14327864,
	Error: 13631488,
	Fatal: 13631488,
}
