package service

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/extra"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
)

var (
	reSay = regexp.MustCompile(`"(.+?)<\d+><(\[.+?])>.+?(say|say_team) "(.+?)"$`)
)

func onPostPingMod() gin.HandlerFunc {
	type pingReq struct {
		ServerName string        `json:"server_name"`
		Name       string        `json:"name"`
		SteamID    steamid.SID64 `json:"steam_id"`
		Reason     string        `json:"reason"`
		Client     int           `json:"client"`
	}
	return func(c *gin.Context) {
		var req pingReq
		if err := c.BindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		for _, c := range config.Discord.ModChannels {
			sendMessage(newMessage(c, fmt.Sprintf("<@&%d> %s", config.Discord.ModRoleID, req.Reason)))
		}
		c.JSON(http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})
	}
}

func onPostLogMessage() gin.HandlerFunc {
	type logReq struct {
		ServerID  string        `json:"server_id"`
		SteamID   steamid.SID64 `json:"steam_id"`
		Name      string        `json:"name"`
		Message   string        `json:"message"`
		TeamSay   bool          `json:"team_say"`
		Timestamp int           `json:"timestamp"`
	}
	return func(c *gin.Context) {
		var req logReq
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("Failed to decode log message: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}
		filtered, word := util.IsFilteredWord(req.Message)
		if filtered {
			addWarning(req.SteamID, warnLanguage)
			for _, c := range config.Relay.ChannelIDs {
				sendMessage(newMessage(c, fmt.Sprintf("<@&%d> Word filter triggered: %s", config.Discord.ModRoleID, word)))
			}
		}
		// [us-2] 76561198017946808 name: message
		msgBody := req.Message
		if req.TeamSay {
			msgBody = "(Team) " + msgBody
		}
		msg := fmt.Sprintf(`[%s] %d **%s** %s`, req.ServerID, req.SteamID, req.Name, msgBody)
		for _, channelID := range config.Relay.ChannelIDs {
			sendMessage(newMessage(channelID, msg))
		}
		c.Status(200)
	}
}

func onSAPIPostServerAuth() gin.HandlerFunc {
	type authReq struct {
		ServerName string `json:"server_name"`
		Key        string `json:"key"`
	}
	type authResp struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
	}
	return func(c *gin.Context) {
		var req authReq
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("Failed to decode auth request: %v", err)
			c.JSON(500, authResp{Status: false})
			return
		}
		srv, err := getServerByName(req.ServerName)
		if err != nil {
			c.JSON(http.StatusNotFound, authResp{Status: false})
			return
		}
		srv.Token = golib.RandomString(40)
		srv.TokenCreatedOn = config.Now()
		if err := SaveServer(&srv); err != nil {
			log.Errorf("Failed to updated server token: %v", err)
			c.JSON(500, authResp{Status: false})
			return
		}
		c.JSON(200, authResp{
			Status: true,
			Token:  srv.Token,
		})
	}
}

func onPostServerCheck() gin.HandlerFunc {
	type checkRequest struct {
		ClientID int    `json:"client_id"`
		SteamID  string `json:"steam_id"`
		IP       net.IP `json:"ip"`
	}
	type checkResponse struct {
		ClientID int           `json:"client_id"`
		SteamID  string        `json:"steam_id"`
		BanType  model.BanType `json:"ban_type"`
		Msg      string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req checkRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		resp := checkResponse{
			ClientID: req.ClientID,
			SteamID:  req.SteamID,
			BanType:  model.Unknown,
			Msg:      "",
		}
		// Check IP first
		banNet, err := getBanNet(req.IP)
		if err != nil {
			c.JSON(500, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			log.Errorf("Could not get ban net results: %v", err)
			return
		}
		if len(banNet) > 0 {
			resp.BanType = model.Banned
			resp.Msg = fmt.Sprintf("Network banned (C: %d)", len(banNet))
			c.JSON(200, resp)
			return
		}
		// Check SteamID
		steamID, err := steamid.ResolveSID64(context.Background(), req.SteamID)
		if err != nil || !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			c.JSON(500, resp)
		}
		ban, err := getBanBySteamID(steamID, false)
		if err != nil {
			if DBErr(err) == errNoResult {
				resp.BanType = model.OK
				c.JSON(200, resp)
				return
			}
			resp.Msg = "Error determining state"
			c.JSON(500, resp)
			return
		}
		resp.BanType = ban.Ban.BanType
		resp.Msg = ban.Ban.ReasonText
		c.JSON(200, resp)
	}
}

func onAPIPostAppeal() gin.HandlerFunc {
	type req struct {
		Email      string `json:"email"`
		AppealText string `json:"appeal_text"`
	}
	return func(c *gin.Context) {
		var app req
		if err := c.BindJSON(&app); err != nil {
			log.Errorf("Received malformed appeal apiBanRequest: %v", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func onAPIPostReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{})
	}
}

func onAPIGetServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := getServers()
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, servers)
	}
}

func onAPIProfile() gin.HandlerFunc {
	type req struct {
		Query string `form:"query"`
	}
	type resp struct {
		Player  *model.Person         `json:"player"`
		Friends []extra.PlayerSummary `json:"friends"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.Bind(&r); err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sid, err := steamid.StringToSID64(r.Query)
		if err != nil {
			sid, err = steamid.ResolveSID64(cx, r.Query)
			if err != nil {
				responseErr(c, http.StatusNotFound, nil)
				return
			}
		}
		person, err := GetOrCreatePersonBySteamID(sid)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		sum, err := extra.PlayerSummaries(cx, []steamid.SID64{sid})
		if err != nil || len(sum) != 1 {
			log.Errorf("Failed to get player summary: %v", err)
			responseErr(c, http.StatusInternalServerError, "Could not fetch summary")
			return
		}
		person.PlayerSummary = &sum[0]
		friendIDs, err := fetchFriends(person.SteamID)
		if err != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch friends")
			return
		}
		friends, err := fetchSummaries(friendIDs)
		if err != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch summaries")
			return
		}
		var response resp
		response.Player = person
		response.Friends = friends
		responseOK(c, http.StatusOK, response)
	}
}
func onAPIGetFilteredWords() gin.HandlerFunc {
	type resp struct {
		Count int      `json:"count"`
		Words []string `json:"words"`
	}
	return func(c *gin.Context) {
		words, err := GetFilteredWords()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
		c.JSON(http.StatusOK, resp{
			Count: len(words),
			Words: words,
		})
	}
}

func onAPIGetStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := getStats()
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		serverStateMu.RLock()
		defer serverStateMu.RUnlock()
		for _, server := range serverState {
			if server.Alive {
				stats.ServersAlive++
			}
		}
		responseOK(c, http.StatusOK, stats)
	}
}

func loadBanMeta(b *model.BannedPerson) {

}

func onAPIGetBanByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		banIDStr := c.Param("ban_id")
		if banIDStr == "" {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		sid, err := strconv.ParseUint(banIDStr, 10, 64)
		if err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}

		ban, err := getBanByBanID(sid, false)
		if err != nil {
			responseErr(c, http.StatusNotFound, nil)
			log.Errorf("Failed to fetch bans")
			return
		}
		loadBanMeta(ban)
		responseOK(c, http.StatusOK, ban)
	}
}

func onAPIGetBans() gin.HandlerFunc {
	type req struct {
		SortDesc bool   `json:"sort_desc"`
		Offset   uint64 `json:"offset"`
		Limit    uint64 `json:"limit"`
		OrderBy  string `json:"order_by"`
		Query    string `json:"query"`
	}
	type resp struct {
		Total int                   `json:"total"`
		Bans  []*model.BannedPerson `json:"bans"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.BindJSON(&r); err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		o := newSearchQueryOpts(r.Query)
		o.Limit = r.Limit
		if o.Limit > 100 {
			o.Limit = 100
		} else if o.Limit <= 0 {
			o.Limit = 100
		}
		o.Offset = r.Offset
		switch o.OrderDesc {
		case true:
			o.OrderDesc = true
		case false:
			fallthrough
		default:
			o.OrderDesc = false
		}
		o.OrderBy = r.OrderBy

		bans, err := GetBans(o)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to fetch bans")
			return
		}
		t, err := GetBansTotal(o)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to fetch ban total")
			return
		}
		responseOK(c, http.StatusOK, resp{
			Total: t,
			Bans:  bans,
		})
	}
}

func onGetServerBan() gin.HandlerFunc {
	type banStateRequest struct {
		SteamID string `json:"steam_id"`
	}
	type banStateResponse struct {
		SteamID string        `json:"steam_id"`
		BanType model.BanType `json:"ban_type"`
		Msg     string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req banStateRequest

		if err := c.BindJSON(&req); err != nil {
			c.JSON(500, banStateResponse{
				SteamID: "",
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		c.JSON(200, gin.H{"status": model.OK})
	}
}

// messageType defines the type of log message being sent
type messageType int

const (
	//TypeLog is a console.log file
	TypeLog messageType = iota
	// TypeStartup is a server start event message
	TypeStartup
	// TypeShutdown is a server start event message
	TypeShutdown
)

// RelayPayload is the container for log/message payloads
type RelayPayload struct {
	Type    messageType `json:"type"`
	Server  string      `json:"server"`
	Message string      `json:"message"`
}

func onPostLogAdd() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RelayPayload
		if err := c.BindJSON(&req); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.Status(http.StatusCreated)

		for _, c := range config.Relay.ChannelIDs {
			sendMessage(newMessage(c, req.Message))
		}
		match := reSay.FindStringSubmatch(req.Message)
		if len(match) != 5 {
			return
		}
		sid64 := steamid.SID3ToSID64(steamid.SID3(match[2]))
		if sid64.Int64() != 76561197960265728 && !sid64.Valid() {
			return
		}
		messageQueue <- discordMessage{
			ChannelID: "",
			Body:      match[4],
		}
	}
}
func onPostBan() gin.HandlerFunc {
	return func(c *gin.Context) {
		var b model.Ban
		c.JSON(http.StatusCreated, b)
	}
}
