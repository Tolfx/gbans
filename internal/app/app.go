// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"go.uber.org/zap"
	"gopkg.in/mxpv/patreon-go.v1"
	"net"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
)

type App struct {
	logFileChan chan *LogFilePayload
	logger      *zap.Logger
	// Current known state of the servers rcon status command
	serverStateStatus   map[string]extra.Status
	serverStateStatusMu *sync.RWMutex
	// Current known state of the servers a2s server info query
	serverStateA2S     map[string]a2s.ServerInfo
	serverStateA2SMu   *sync.RWMutex
	masterServerList   []model.ServerLocation
	masterServerListMu *sync.RWMutex
	discordSendMsg     chan discordPayload
	warningChan        chan newUserWarning
	notificationChan   chan notificationPayload
	serverStateMu      *sync.RWMutex
	serverState        model.ServerStateCollection

	bannedGroupMembers   map[steamid.GID]steamid.Collection
	bannedGroupMembersMu *sync.RWMutex
	ctx                  context.Context
	store                store.Store
	patreon              *patreon.Client
	patreonMu            *sync.RWMutex
	patreonCampaigns     []patreon.Campaign
	patreonPledges       []patreon.Pledge
	patreonUsers         map[string]*patreon.User
}

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion = "master"
)

type userWarning struct {
	WarnReason    model.Reason
	Message       string
	Matched       string
	MatchedFilter *model.Filter
	CreatedOn     time.Time
}

type discordPayload struct {
	channelId string
	embed     *discordgo.MessageEmbed
}

func New(ctx context.Context, logger *zap.Logger) *App {
	app := App{
		logger:               logger,
		logFileChan:          make(chan *LogFilePayload, 10),
		serverStateStatus:    map[string]extra.Status{},
		serverStateStatusMu:  &sync.RWMutex{},
		serverStateA2S:       map[string]a2s.ServerInfo{},
		serverStateA2SMu:     &sync.RWMutex{},
		masterServerList:     []model.ServerLocation{},
		masterServerListMu:   &sync.RWMutex{},
		discordSendMsg:       make(chan discordPayload, 5),
		warningChan:          make(chan newUserWarning),
		notificationChan:     make(chan notificationPayload, 5),
		serverStateMu:        &sync.RWMutex{},
		serverState:          model.ServerStateCollection{},
		bannedGroupMembers:   map[steamid.GID]steamid.Collection{},
		bannedGroupMembersMu: &sync.RWMutex{},
		patreonMu:            &sync.RWMutex{},
		ctx:                  ctx,
	}
	return &app
}

func firstTimeSetup(ctx context.Context, logger *zap.Logger, db store.Store) error {
	if !config.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var owner model.Person
	if errRootUser := db.GetPersonBySteamID(localCtx, config.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, store.ErrNoResult) {
			return errors.Wrapf(errRootUser, "Failed first time setup")
		}
		logger.Info("Performing initial setup")
		newOwner := model.NewPerson(config.General.Owner)
		newOwner.PermissionLevel = model.PAdmin
		if errSave := db.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Wrap(errSave, "Failed to create admin user")
		}
		newsEntry := model.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}
		if errSave := db.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample news entry")
		}
		server := model.NewServer("server-1", "127.0.0.1", 27015)
		server.CC = "jp"
		server.RCON = "example_rcon"
		server.Latitude = 35.652832
		server.Longitude = 139.839478
		server.ServerNameLong = "Example Server"
		server.LogSecret = 12345678
		server.Region = "asia"
		if errSave := db.SaveServer(localCtx, &server); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample server entry")
		}
		var page wiki.Page
		page.BodyMD = "# Welcome to the wiki"
		page.UpdatedOn = time.Now()
		page.CreatedOn = time.Now()
		page.Revision = 1
		page.Slug = wiki.RootSlug
		if errSave := db.SaveWikiPage(localCtx, &page); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample wiki entry")
		}
	}
	return nil
}

// Start is the main application entry point
func (app *App) Start() error {
	dbStore, dbErr := store.New(app.ctx, app.logger, config.DB.DSN)
	if dbErr != nil {
		return errors.Wrapf(dbErr, "Failed to setup store")
	}
	defer func() {
		if errClose := dbStore.Close(); errClose != nil {
			app.logger.Error("Error cleanly closing app", zap.Error(errClose))
		}
	}()
	app.store = dbStore

	if setupErr := firstTimeSetup(app.ctx, app.logger, app.store); setupErr != nil {
		app.logger.Fatal("Failed to do first time setup", zap.Error(setupErr))
	}

	patreonClient, errPatreon := thirdparty.NewPatreonClient(app.ctx, app.logger, dbStore)
	if errPatreon == nil {
		app.patreon = patreonClient
	}

	webService, errWeb := NewWeb(app)
	if errWeb != nil {
		return errors.Wrapf(errWeb, "Failed to setup web")
	}

	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		if errNetBans := initNetBans(); errNetBans != nil {
			return errors.Wrap(errNetBans, "Failed to load net bans")
		}
	} else {
		app.logger.Warn("External Network ban lists not enabled")
	}

	// Start the discord service
	if config.Discord.Enabled {
		go app.initDiscord(app.ctx, app.discordSendMsg)
	} else {
		app.logger.Warn("discord bot not enabled")
	}

	// Start the background goroutine workers
	app.initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		if errFilter := initFilters(app.ctx, dbStore); errFilter != nil {
			return errors.Wrap(errFilter, "Failed to load filters")
		}
		app.logger.Info("Loaded filter list", zap.Int("count", len(wordFilters)))
	}
	if errSend := app.sendNotification(notificationPayload{
		minPerms: model.PAdmin,
		sids:     nil,
		severity: model.SeverityInfo,
		message:  "App start",
		link:     "",
	}); errSend != nil {
		app.logger.Error("Failed to send notification", zap.Error(errSend))
	}
	// Start & block, listening on the HTTP server
	if errHttpListen := webService.ListenAndServe(app.ctx); errHttpListen != nil {
		return errors.Wrapf(errHttpListen, "Error shutting down service")
	}
	return nil
}

type newUserWarning struct {
	ServerEvent model.ServerEvent
	Message     string
	userWarning
}

// warnWorker handles tracking and applying warnings based on incoming events
func (app *App) warnWorker() {
	warnings := map[steamid.SID64][]userWarning{}
	eventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(eventChan, []logparse.EventType{logparse.Say, logparse.SayTeam}); errRegister != nil {
		app.logger.Fatal("Failed to register event reader", zap.Error(errRegister))
	}
	ticker := time.NewTicker(1 * time.Second)
	warningHandler := func() {
		for {
			select {
			case now := <-ticker.C:
				for steamId := range warnings {
					for warnIdx, warning := range warnings[steamId] {
						if now.Sub(warning.CreatedOn) > config.General.WarningTimeout {
							if len(warnings[steamId]) > 1 {
								warnings[steamId] = append(warnings[steamId][:warnIdx], warnings[steamId][warnIdx+1])
							} else {
								delete(warnings, steamId)
							}
						}
					}
				}
			case newWarn := <-app.warningChan:
				steamId := newWarn.ServerEvent.Source.SteamID
				if !steamId.Valid() {
					continue
				}
				if newWarn.ServerEvent.Server.ServerID != 408 {
					continue
				}
				app.logger.Info("User triggered word filter",
					zap.String("matched", newWarn.Matched),
					zap.String("message", newWarn.Message),
					zap.Int64("filter_id", newWarn.MatchedFilter.FilterID))
				_, found := warnings[steamId]
				if !found {
					warnings[steamId] = []userWarning{}
				}
				warnings[steamId] = append(warnings[steamId], newWarn.userWarning)
				warnNotice := &discordgo.MessageEmbed{
					URL:   config.ExtURL("/profiles/%d", steamId),
					Type:  discordgo.EmbedTypeRich,
					Title: fmt.Sprintf("Language Warning (#%d/%d)", len(warnings[steamId]), config.General.WarningLimit),
					Color: int(orange),
					Image: &discordgo.MessageEmbedImage{URL: newWarn.ServerEvent.Source.AvatarFull},
				}
				addField(warnNotice, app.logger, "Matched", newWarn.Matched)
				addField(warnNotice, app.logger, "Message", newWarn.userWarning.Message)
				if len(warnings[steamId]) > config.General.WarningLimit {
					app.logger.Info("Warn limit exceeded",
						zap.Int64("sid64", steamId.Int64()),
						zap.Int("count", len(warnings[steamId])))
					var errBan error
					var banSteam model.BanSteam
					if errNewBan := NewBanSteam(model.StringSID(config.General.Owner.String()),
						model.StringSID(steamId.String()),
						model.Duration(config.General.WarningExceededDurationValue),
						newWarn.WarnReason,
						"",
						"Automatic warning ban",
						model.System,
						0,
						model.NoComm,
						&banSteam); errNewBan != nil {
						app.logger.Error("Failed to create warning ban", zap.Error(errNewBan))
						continue
					}
					switch config.General.WarningExceededAction {
					case config.Gag:
						banSteam.BanType = model.NoComm
						errBan = app.BanSteam(app.ctx, &banSteam)
					case config.Ban:
						banSteam.BanType = model.Banned
						errBan = app.BanSteam(app.ctx, &banSteam)
					case config.Kick:
						var playerInfo model.PlayerInfo
						errBan = app.Kick(app.ctx, model.System, model.StringSID(steamId.String()),
							model.StringSID(config.General.Owner.String()), newWarn.WarnReason, &playerInfo)
					}
					if errBan != nil {
						app.logger.Error("Failed to apply warning action",
							zap.Error(errBan),
							zap.String("action", string(config.General.WarningExceededAction)))
					}
					addField(warnNotice, app.logger, "Name", newWarn.ServerEvent.Source.PersonaName)
					expIn := "Permanent"
					expAt := "Permanent"
					if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
						expIn = config.FmtDuration(banSteam.ValidUntil)
						expAt = config.FmtTimeShort(banSteam.ValidUntil)
					}
					addField(warnNotice, app.logger, "Expires In", expIn)
					addField(warnNotice, app.logger, "Expires At", expAt)
				} else {
					msg := fmt.Sprintf("[WARN #%d] Please refrain from using slurs/toxicity (see: rules & MOTD). "+
						"Further offenses will result in mutes/bans", len(warnings[steamId]))
					if errPSay := app.PSay(app.ctx, 0, model.StringSID(steamId.String()), msg, &newWarn.ServerEvent.Server); errPSay != nil {
						app.logger.Error("Failed to send user warning psay message", zap.Error(errPSay))
					}
				}
				addField(warnNotice, app.logger, "Pattern", newWarn.MatchedFilter.Pattern)
				addFieldsSteamID(warnNotice, app.logger, steamId)
				addFieldInt64Inline(warnNotice, app.logger, "Filter ID", newWarn.MatchedFilter.FilterID)
				addFieldInline(warnNotice, app.logger, "Server", newWarn.ServerEvent.Server.ServerNameShort)
				app.sendDiscordPayload(discordPayload{
					channelId: config.Discord.ModLogChannelId,
					embed:     warnNotice,
				})
				newWarn.MatchedFilter.TriggerCount++
				if errSave := app.store.SaveFilter(app.ctx, newWarn.MatchedFilter); errSave != nil {
					app.logger.Error("Failed to update filter trigger count", zap.Error(errSave))
				}
			case <-app.ctx.Done():
				return
			}
		}
	}

	go warningHandler()

	for {
		select {
		case serverEvent := <-eventChan:
			msg, found := serverEvent.MetaData["msg"].(string)
			if !found {
				continue
			}
			matchedWord, matchedFilter := findFilteredWordMatch(msg)
			if matchedFilter != nil {
				app.warningChan <- newUserWarning{
					ServerEvent: serverEvent,
					userWarning: userWarning{
						WarnReason:    model.Language,
						Message:       msg,
						Matched:       matchedWord,
						MatchedFilter: matchedFilter,
						CreatedOn:     config.Now(),
					},
				}
			}
		case <-app.ctx.Done():
			return
		}
	}
}

func (app *App) matchSummarizer() {
	eventChan := make(chan model.ServerEvent)
	if errReg := event.Consume(eventChan, []logparse.EventType{logparse.Any}); errReg != nil {
		app.logger.Error("logWriter Tried to register duplicate reader channel", zap.Error(errReg))
	}
	matches := map[int]model.Match{}

	var curServer model.Server
	for {
		select {
		case evt := <-eventChan:
			match, found := matches[evt.Server.ServerID]
			if !found && evt.EventType != logparse.MapLoad {
				// Wait for new map
				continue
			}
			if evt.EventType == logparse.LogStart {
				app.logger.Info("New match created (new game)", zap.String("server", evt.Server.ServerNameShort))
				matches[evt.Server.ServerID] = model.NewMatch(app.logger, evt.Server.ServerID, evt.Server.ServerNameLong)
			}
			// Apply the update before any secondary side effects trigger
			if errApply := match.Apply(evt); errApply != nil {
				app.logger.Error("Error applying event",
					zap.String("server", evt.Server.ServerNameShort),
					zap.Error(errApply))
			}
			switch evt.EventType {
			case logparse.LogStop:
				fallthrough
			case logparse.WGameOver:
				go func(completeMatch model.Match) {
					if errSave := app.store.MatchSave(app.ctx, &completeMatch); errSave != nil {
						app.logger.Error("Failed to save match",
							zap.String("server", evt.Server.ServerNameShort), zap.Error(errSave))
					} else {
						sendDiscordMatchResults(app, curServer, completeMatch)
					}
				}(match)
				delete(matches, evt.Server.ServerID)
			}
		case <-app.ctx.Done():
			return
		}
	}
}

func sendDiscordMatchResults(app *App, server model.Server, match model.Match) {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       fmt.Sprintf("Match #%d - %s - %s", match.MatchID, server.ServerNameShort, match.MapName),
		Description: "Match results",
		Color:       int(green),
		URL:         config.ExtURL("/log/%d", match.MatchID),
	}
	redScore := 0
	bluScore := 0
	for _, round := range match.Rounds {
		redScore += round.Score.Red
		bluScore += round.Score.Blu
	}

	found := 0
	for _, teamStats := range match.TeamSums {
		addFieldInline(embed, app.logger, fmt.Sprintf("%s Kills", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Kills))
		addFieldInline(embed, app.logger, fmt.Sprintf("%s Damage", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Damage))
		addFieldInline(embed, app.logger, fmt.Sprintf("%s Ubers/Drops", teamStats.Team.String()), fmt.Sprintf("%d/%d", teamStats.Charges, teamStats.Drops))
		found++
	}
	addFieldInline(embed, app.logger, "Red Score", fmt.Sprintf("%d", redScore))
	addFieldInline(embed, app.logger, "Blu Score", fmt.Sprintf("%d", bluScore))
	addFieldInline(embed, app.logger, "Duration", fmt.Sprintf("%.2f Minutes", time.Since(match.CreatedOn).Minutes()))
	app.sendDiscordPayload(discordPayload{channelId: config.Discord.LogChannelID, embed: embed})
}

func (app *App) playerMessageWriter() {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(serverEventChan, []logparse.EventType{
		logparse.Say,
		logparse.SayTeam,
	}); errRegister != nil {
		app.logger.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))
		return
	}
	for {
		select {
		case <-app.ctx.Done():
			return
		case evt := <-serverEventChan:
			switch evt.EventType {
			case logparse.Say:
				fallthrough
			case logparse.SayTeam:
				body := evt.GetValueString("msg")
				if body == "" {
					app.logger.Warn("Empty person message body, skipping")
					continue
				}
				msg := model.PersonMessage{
					SteamId:     evt.Source.SteamID,
					PersonaName: evt.Source.PersonaName,
					ServerName:  evt.Server.ServerNameLong,
					ServerId:    evt.Server.ServerID,
					Body:        body,
					Team:        evt.EventType == logparse.SayTeam,
					CreatedOn:   evt.CreatedOn,
				}
				lCtx, cancel := context.WithTimeout(app.ctx, time.Second*5)
				if errChat := app.store.AddChatHistory(lCtx, &msg); errChat != nil {
					app.logger.Error("Failed to add chat history", zap.Error(errChat))
				}
				cancel()
				app.logger.Debug("Saved user chat message", zap.String("message", msg.Body))
			}
		}
	}
}

func (app *App) playerConnectionWriter() {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(serverEventChan, []logparse.EventType{logparse.Connected}); errRegister != nil {
		app.logger.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))
		return
	}
	for {
		select {
		case <-app.ctx.Done():
			return
		case evt := <-serverEventChan:
			addr := evt.GetValueString("address")
			if addr == "" {
				app.logger.Warn("Empty person message body, skipping")
				continue
			}
			parsedAddr := net.ParseIP(addr)
			if parsedAddr == nil {
				app.logger.Warn("Received invalid address", zap.String("addr", addr))
				continue
			}
			msg := model.PersonConnection{
				IPAddr:      parsedAddr,
				SteamId:     evt.Source.SteamID,
				PersonaName: evt.Source.PersonaName,
				CreatedOn:   evt.CreatedOn,
			}
			lCtx, cancel := context.WithTimeout(app.ctx, time.Second*5)
			if errChat := app.store.AddConnectionHistory(lCtx, &msg); errChat != nil {
				app.logger.Error("Failed to add connection history", zap.Error(errChat))
			}
			cancel()
		}
	}
}

type LogFilePayload struct {
	Server model.Server
	Lines  []string
	Map    string
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader
func (app *App) logReader() {
	var file *os.File
	if config.Debug.WriteUnhandledLogEvents {
		var errCreateFile error
		file, errCreateFile = os.Create("./unhandled_messages.log")
		if errCreateFile != nil {
			app.logger.Fatal("Failed to open debug message log", zap.Error(errCreateFile))
		}
		defer func() {
			if errClose := file.Close(); errClose != nil {
				app.logger.Error("Failed to close unhandled_messages.log", zap.Error(errClose))
			}
		}()
	}
	playerStateCache := newPlayerCache(app.logger)
	for {
		select {
		case logFile := <-app.logFileChan:
			emitted := 0
			failed := 0
			unknown := 0
			ignored := 0
			for _, logLine := range logFile.Lines {
				var serverEvent model.ServerEvent
				if errLogServerEvent := logToServerEvent(app.ctx, app.logger, logFile.Server, logLine, app.store,
					playerStateCache, &serverEvent); errLogServerEvent != nil {
					app.logger.Error("Failed to parse log line", zap.Error(errLogServerEvent))
					failed++
					continue
				}
				if serverEvent.EventType == logparse.IgnoredMsg {
					ignored++
					continue
				} else if serverEvent.EventType == logparse.UnknownMsg {
					unknown++
					if config.Debug.WriteUnhandledLogEvents {
						if _, errWrite := file.WriteString(logLine + "\n"); errWrite != nil {
							app.logger.Error("Failed to write debug log", zap.Error(errWrite))
						}
					}
				}
				event.Emit(serverEvent)
				emitted++
			}
			app.logger.Debug("Completed emitting logfile events",
				zap.Int("ok", emitted), zap.Int("failed", failed),
				zap.Int("unknown", unknown), zap.Int("ignored", ignored))
		case <-app.ctx.Done():
			app.logger.Debug("logReader shutting down")
			return
		}
	}
}

func initFilters(ctx context.Context, database store.FilterStore) error {
	// TODO load external lists via http
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()
	words, errGetFilters := database.GetFilters(localCtx)
	if errGetFilters != nil {
		if errors.Is(errGetFilters, store.ErrNoResult) {
			return nil
		}
		return errGetFilters
	}
	importFilteredWords(words)
	return nil
}

func (app *App) initWorkers() {
	statusUpdateFreq, errDuration := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errDuration != nil {
		app.logger.Fatal("Failed to parse server_status_update_freq", zap.Error(errDuration))
	}
	masterUpdateFreq, errParseMasterUpdateFreq := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errParseMasterUpdateFreq != nil {
		app.logger.Fatal("Failed to parse master_server_status_update_freq", zap.Error(errParseMasterUpdateFreq))
	}
	go app.patreonUpdater()
	go app.banSweeper()
	go app.serverA2SStatusUpdater(statusUpdateFreq)
	go app.serverRCONStatusUpdater(statusUpdateFreq)
	go app.serverStateRefresher(statusUpdateFreq)
	go app.profileUpdater()
	go app.warnWorker()
	go app.logReader()
	go app.initLogSrc()
	go app.logMetricsConsumer()
	go app.matchSummarizer()
	go app.playerMessageWriter()
	go app.playerConnectionWriter()
	go app.steamGroupMembershipUpdater()
	go app.localStatUpdater()
	go app.masterServerListUpdater(masterUpdateFreq)
	go app.cleanupTasks()
	go app.showReportMeta()
	go app.notificationSender()
	go app.demoCleaner()
}

// UDP log sink
func (app *App) initLogSrc() {
	logSrc, errLogSrc := newRemoteSrcdsLogSource(app.ctx, app.logger, config.Log.SrcdsLogAddr, app.store)
	if errLogSrc != nil {
		app.logger.Fatal("Failed to setup udp log src", zap.Error(errLogSrc))
	}
	logSrc.start(app.store)
}

func (app *App) sendUserNotification(pl notificationPayload) {
	select {
	case app.notificationChan <- pl:
	default:
		app.logger.Error("Failed to write user notification payload, channel full")
	}
}

func (app *App) initDiscord(ctx context.Context, botSendMessageChan chan discordPayload) {
	if config.Discord.Token != "" {
		session, sessionErr := NewDiscord(app)
		if sessionErr != nil {
			app.logger.Fatal("Failed to setup session", zap.Error(sessionErr))
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case payload := <-botSendMessageChan:
					if !session.Connected.Load() {
						app.logger.Warn("Skipped payload for unconnected discord")
						continue
					}
					if errSend := session.SendEmbed(payload.channelId, payload.embed); errSend != nil {
						app.logger.Error("Failed to send discord payload", zap.Error(errSend))
					}
				}
			}
		}()
		if errSessionStart := session.Start(ctx, config.Discord.Token); errSessionStart != nil {
			app.logger.Error("discord returned error", zap.Error(errSessionStart))
		}
	} else {
		app.logger.Fatal("discord enabled, but bot token invalid")
	}
}

func initNetBans() error {
	for _, banList := range config.Net.Sources {
		if _, errImport := thirdparty.Import(banList); errImport != nil {
			return errImport
		}
	}
	return nil
}

// validateLink is used in the case of discord origin actions that require mapping the
// discord member ID to a SteamID so that we can track its use and apply permissions, etc.
//
// This function will replace the discord member id value in the target field with
// the found SteamID, if any.
//func validateLink(ctx context.Context, database store.Store, sourceID action.Author, target *action.Author) error {
//	var p model.Person
//	if errGetPerson := database.GetPersonByDiscordID(ctx, string(sourceID), &p); errGetPerson != nil {
//		if errGetPerson == store.ErrNoResult {
//			return consts.ErrUnlinkedAccount
//		}
//		return consts.ErrInternal
//	}
//	*target = action.Author(p.SteamID.String())
//	return nil
//}
