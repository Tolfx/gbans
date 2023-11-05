// Package thirdparty implements functionality for communicating and parsing external or 3rd party data sources.
package thirdparty

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

var (
	networks []*net.IPNet    //nolint:gochecknoglobals
	steamids []steamid.SID64 //nolint:gochecknoglobals
)

func containsSID(sid steamid.SID64) bool {
	for _, s := range steamids {
		if s.Int64() == sid.Int64() {
			return true
		}
	}

	return false
}

func containsIP(ip net.IP) bool {
	for _, b := range networks {
		if b.Contains(ip) {
			return true
		}
	}

	return false
}

// BanListType is the type or source of a ban list.
type BanListType string

const (
	// CIDR formatted list.
	CIDR BanListType = "cidr"
	// ValveNet is the srcds network ban list format.
	ValveNet BanListType = "valve_net"
	// ValveSID is the srcds steamid ban list format.
	ValveSID BanListType = "valve_steamid"
	// TF2BD sources ban list.
	TF2BD BanListType = "tf2bd"
)

// BanList holds details to load a ban lost.
type BanList struct {
	URL  string      `mapstructure:"url"`
	Name string      `mapstructure:"name"`
	Type BanListType `mapstructure:"type"`
}

// Import is used to download and load block lists into memory.
func Import(ctx context.Context, list BanList, cachePath string, maxAge time.Duration) (int, error) {
	if !golib.Exists(cachePath) {
		if errMkDir := os.MkdirAll(cachePath, 0o755); errMkDir != nil {
			return 0, errors.Wrapf(errMkDir, "Failed to create cache dir (%s): %v", cachePath, errMkDir)
		}
	}

	filePath := path.Join(cachePath, list.Name)

	expired := false

	if golib.Exists(filePath) {
		fileInfo, errStat := os.Stat(filePath)
		if errStat != nil {
			return 0, errors.Wrapf(errStat, "Failed to stat cached file")
		}

		if time.Since(fileInfo.ModTime()) > maxAge {
			expired = true
		}
	} else {
		expired = true
	}

	if expired {
		if errDownload := download(ctx, list.URL, filePath); errDownload != nil {
			return 0, errors.Wrapf(errDownload, "Failed to download net ban list")
		}
	}

	body, errReadFile := os.ReadFile(filePath)
	if errReadFile != nil {
		return 0, errors.Wrapf(errReadFile, "Failed to read file")
	}

	count, errLoadBody := load(body, list.Type)
	if errLoadBody != nil {
		return 0, errors.Wrapf(errLoadBody, "Failed to load list")
	}

	return count, nil
}

func download(ctx context.Context, url string, savePath string) error {
	client := util.NewHTTPClient()

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errReq != nil {
		return errors.Wrapf(errReq, "Failed to create request")
	}

	response, errQuery := client.Do(req)
	if errQuery != nil {
		return errors.Wrapf(errQuery, "Failed to perform request")
	}

	outFile, errCreate := os.Create(savePath)
	if errCreate != nil {
		return errors.Wrapf(errQuery, "Failed to create output file")
	}

	_, errCopy := io.Copy(outFile, response.Body)
	if errCopy != nil {
		return errors.Wrapf(errCopy, "Failed to copy response body")
	}

	if errClose := response.Body.Close(); errClose != nil {
		return errors.Wrapf(errClose, "Failed to close response")
	}

	return nil
}

func load(src []byte, listType BanListType) (int, error) {
	switch listType {
	case CIDR:
		nets, errParseCIDR := parseCIDR(src)
		if errParseCIDR != nil {
			return 0, errParseCIDR
		}

		return addNets(nets), nil
	case ValveNet:
		nets, errParseValveNet := parseValveNet(src)
		if errParseValveNet != nil {
			return 0, errParseValveNet
		}

		return addNets(nets), nil
	case ValveSID:
		ids, errParseValveSID := parseValveSID(src)
		if errParseValveSID != nil {
			return 0, errParseValveSID
		}

		return addSIDs(ids), nil
	case TF2BD:
		ids, errParseBD := parseTF2BD(src)
		if errParseBD != nil {
			return 0, errParseBD
		}

		return addSIDs(ids), nil
	default:
		return 0, errors.Errorf("Unimplemented list type: %v", listType)
	}
}

func addNets(networks []*net.IPNet) int {
	count := 0

	for _, network := range networks {
		if !containsIP(network.IP) {
			networks = append(networks, network)
			count++
		}
	}

	return count
}

func addSIDs(steamIds steamid.Collection) int {
	count := 0

	for _, sid64 := range steamIds {
		if !containsSID(sid64) {
			steamids = append(steamids, sid64)
			count++
		}
	}

	return count
}

func parseCIDR(src []byte) ([]*net.IPNet, error) {
	var nets []*net.IPNet //nolint:prealloc

	for _, line := range strings.Split(string(src), "\n") {
		if line == "" {
			continue
		}

		_, ipNet, errParseCIDR := net.ParseCIDR(line)
		if errParseCIDR != nil {
			continue
		}

		nets = append(nets, ipNet)
	}

	return nets, nil
}
