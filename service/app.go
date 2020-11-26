package service

import (
	"context"
	"encoding/gob"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/bot"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/external"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/gbans/util"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	BuildVersion  = "master"
	router        *gin.Engine
	templates     map[string]*template.Template
	routes        map[Route]string
	ctx           context.Context
	serverStateMu *sync.RWMutex
	serverState   map[string]ServerState
	warnings      map[steamid.SID64][]UserWarning
	warningsMu    *sync.RWMutex
)

type WarnReason int

const (
	warnLanguage WarnReason = iota
)

type UserWarning struct {
	WarnReason WarnReason
	CreatedOn  time.Time
}

// warnWorker will periodically flush out warning older than `config.General.WarningTimeout`
func warnWorker() {
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			now := time.Now().UTC()
			warningsMu.Lock()
			for k := range warnings {
				for i, w := range warnings[k] {
					if now.Sub(w.CreatedOn) > config.General.WarningTimeout {
						if len(warnings[k]) > 1 {
							warnings[k] = append(warnings[k][:i], warnings[k][i+1])
						} else {
							warnings[k] = nil
						}
					}
					if len(warnings[k]) == 0 {
						delete(warnings, k)
					}
				}
			}
			warningsMu.Unlock()
		}
	}
}

// addWarning records a user warning into memory. This is not persistent, so application
// restarts will wipe the users history.
//
// Warning are flushed once they reach N age as defined by `config.General.WarningTimeout
func addWarning(sid64 steamid.SID64, reason WarnReason) {
	warningsMu.Lock()
	defer warningsMu.Unlock()
	_, found := warnings[sid64]
	if !found {
		warnings[sid64] = []UserWarning{}
	}
	warnings[sid64] = append(warnings[sid64], UserWarning{
		WarnReason: reason,
		CreatedOn:  time.Now(),
	})
	if len(warnings[sid64]) >= config.General.WarningLimit {
		Ban(ctx, sid64.String(), "system", 0,
			nil, model.Banned, model.WarningsExceeded, "Warning limit exceeded", model.System)
	}
}

type gameType string

const (
	unknown gameType = "Unknown"
	tf2     gameType = "Team Fortress 2"
	cs      gameType = "Counter-Strike"
	csgo    gameType = "Counter-Strike: Global Offensive"
)

type ServerState struct {
	Addr     string
	Port     int
	Slots    int
	GameType gameType
	A2SInfo  *a2s.ServerInfo
	extra.Status
	// TODO Find better way to track this
	Alive bool
}

func (s ServerState) OS() template.HTML {
	switch s.A2SInfo.ServerOS {
	case a2s.ServerOS_Linux:
		return "linux"
	case a2s.ServerOS_Windows:
		return "windows"
	case a2s.ServerOS_Mac:
		return "mac"
	default:
		return "unknown"
	}
}

func (s ServerState) VacStatus() template.HTML {
	if s.A2SInfo.VAC {
		return "on"
	}
	return "off"
}

func init() {
	gob.Register(Flash{})
	warningsMu = &sync.RWMutex{}
	warnings = make(map[steamid.SID64][]UserWarning)
	templates = make(map[string]*template.Template)
	serverState = make(map[string]ServerState)
	serverStateMu = &sync.RWMutex{}
	ctx = context.Background()
	router = gin.New()
}

// Start is the main application entry point
//
func Start() {
	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		initNetBans()
	} else {
		log.Warnf("External Network ban lists not enabled")
	}
	// Load the HTML templated into memory
	initTemplates()
	// Setup the HTTP router
	initRouter()
	// Setup the storage backend
	initStore()
	// Start the discord service
	if config.Discord.Enabled {
		initDiscord()
	} else {
		log.Warnf("Discord bot not enabled")
	}

	// Start the background goroutine workers
	initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters()
	}

	// Start the HTTP server
	initHTTP()
}

func initFilters() {
	// TODO load external lists via http
	words, err := store.GetFilteredWords()
	if err != nil {
		log.Fatal("Failed to load word list")
	}
	util.ImportFilteredWords(words)
	log.Debugf("Loaded %d filtered words", len(words))
}

func initStore() {
	store.Init(config.DB.Path)
}

func initWorkers() {
	go banSweeper()
	go serverStateUpdater()
	go profileUpdater()
	go updateSearchIndex()
	go warnWorker()
}

func initDiscord() {
	if config.Discord.Token != "" {
		go bot.Start(ctx, config.Discord.Token, config.Discord.ModChannels)
	} else {
		log.Fatalf("Discord enabled, but bot token invalid")
	}
}

func initNetBans() {
	for _, list := range config.Net.Sources {
		if err := external.Import(list); err != nil {
			log.Errorf("Failed to import list: %v", err)
		}
	}
}

func Ban(ctx context.Context, sidStr string, author string, duration time.Duration, ip net.IP,
	banType model.BanType, reason model.Reason, reasonText string, source model.BanSource) error {
	sid, err := steamid.StringToSID64(sidStr)
	if err != nil || !sid.Valid() {
		return errors.Errorf("Failed to get steam id from; %s", sidStr)
	}
	aid, err := steamid.StringToSID64(author)
	if err != nil || !aid.Valid() {
		return errors.Errorf("Failed to get steam id from; %s", sidStr)
	}
	var until int64
	if duration.Seconds() != 0 {
		until = time.Now().Add(duration).Unix()
	}
	ban := model.Ban{
		SteamID:    sid,
		AuthorID:   aid,
		BanType:    banType,
		Reason:     reason,
		ReasonText: reasonText,
		Note:       "naughty",
		Until:      until,
		Source:     source,
		CreatedOn:  time.Now().Unix(),
		UpdatedOn:  time.Now().Unix(),
	}
	if err := store.SaveBan(&ban); err != nil {
		return store.DBErr(err)
	}
	servers, err := store.GetServers()
	if err != nil {
		log.Errorf("Failed to get server for ban propagation")
	}
	QueryRCON(ctx, servers, "gb_kick ")
	return nil
}

func queryInt(c *gin.Context, key string) int {
	valStr := c.Query(key)
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil {
		log.Panicf("Failed to parse query (Use a validator): \"%v\"", valStr)
	}
	return int(val)
}
