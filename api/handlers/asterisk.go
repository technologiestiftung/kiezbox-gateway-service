package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

func uci_get(key string) (string, error) {
	output, err := exec.Command("uci", "get", key).Output()
	if err != nil {
		slog.Error("uci_get error", "err", err)
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

type SipUser struct {
	username string
	password string
	callerid string
}

func idToExt(id int64) string {
	return fmt.Sprintf("%s%04d", defaultUserPrefix, id)
}

func idToCid(id int64) string {
	//TODO: callerid can't contain spaces (or probably other special characters) currently
	// as url.Values.Encode() encodes it like "application/x-www-form-urlencoded"
	// but the asterisk curl backend despite documentation suggesting it does not parse this encoding correctly
	// https://docs.asterisk.org/Configuration/Interfaces/Back-end-Database-and-Realtime-Connectivity/cURL/
	// at least '+' chars as spaces are not decoded correctly and everything before the last '<' char is used as first part of the called id
	basenr, err := uci_get("kb.main.trunk_base")
	if err != nil {
		return fmt.Sprintf("2%d<2%d>", id, id)
	} else {
		return fmt.Sprintf("00%s2%d<00%s>", basenr, id, basenr)
	}
}

func getSession(extension string) (*sipSession, error) {
	files, err := os.ReadDir(defaultSessionPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read from session directory %s: %v", defaultSessionPath, err)
	}
	for _, file := range files {
		filePath := filepath.Join(defaultSessionPath, file.Name())
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			session_content, err := os.ReadFile(filePath)
			if err != nil {
				slog.Error("Failed to read file", "file", filePath, "err", err)
				continue
			}
			var session sipSession
			err = json.Unmarshal(session_content, &session)
			if err != nil {
				slog.Error("Error unmarshaling JSON", "err", err)
				continue
			}
			if idToExt(session.Extension) == extension {
				return &session, nil
			}
		}
	}
	return nil, fmt.Errorf("extension %s not found", extension)
}

func getSessions(pattern string) (*[]sipSession, error) {
	files, err := os.ReadDir(defaultSessionPath)
	var sessions []sipSession
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	} else {
		found := false
		for _, file := range files {
			filePath := filepath.Join(defaultSessionPath, file.Name())
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				session_content, err := os.ReadFile(filePath)
				if err != nil {
					slog.Error("Failed to read file", "file", filePath, "err", err)
					continue
				}
				var session sipSession
				err = json.Unmarshal(session_content, &session)
				if err != nil {
					slog.Error("Error unmarshaling JSON", "err", err)
					continue
				}
				if re.MatchString(fmt.Sprintf("%s%04d", defaultUserPrefix, session.Extension)) {
					found = true
					sessions = append(sessions, session)
				}
			}
		}
		if found {
			return &sessions, nil
		} else {
			return nil, fmt.Errorf("no session found matching pattern %s", pattern)
		}
	}
}

func Asterisk(c *gin.Context) {
	SipEndpoint := map[string]string{
		"type":                 "endpoint",
		"moh_suggest":          "default",
		"context":              "from-extensions",
		"inband_progress":      "no",
		"rtp_timeout":          "120",
		"direct_media":         "no",
		"dtmf_mode":            "rfc4733",
		"device_state_busy_at": "1",
		"disallow":             "all",
		"transport":            "wss_transport",
		"allow":                "opus,ulaw,vp9,vp8,h264",
		"webrtc":               "yes",
	}
	SipAuth := map[string]string{
		"type":      "auth",
		"auth_type": "userpass",
	}
	SipAor := map[string]string{
		"type":              "aor",
		"max_contacts":      "1",
		"qualify_frequency": "120",
		"remove_existing":   "yes",
	}
	is_single := false
	if c.Param("singlemulti") == "single" {
		is_single = true
	}
	slog.Info("Request received", "pstype", c.Param("pstype"), "single", is_single)
	var sessions []sipSession
	if is_single {
		id, found := c.GetPostForm("id")
		if found {
			session, err := getSession(id)
			if err == nil {
				sessions = append(sessions, *session)
			} else {
				slog.Warn("ID not found", "id", id)
				c.String(http.StatusNotFound, "ID %s not found", id)
				return
			}
		} else {
			slog.Warn("Parameter `id` not set")
			c.String(http.StatusBadRequest, "Parameter `id` not set")
			return
		}
	} else {
		//TODO: check if we need to implement any other parameters, I've seen requests to `mailboxes%20!%3D=` in the logs
		idLike, found := c.GetPostForm("id LIKE")
		if found {
			idLikeRegex := "^" + strings.ReplaceAll(strings.ReplaceAll(idLike, "%", ".*"), "_", ".") + "$"
			matched_sessions, err := getSessions(idLikeRegex)
			if err == nil {
				sessions = append(sessions, *matched_sessions...)
			} else {
				slog.Warn("ID LIKE not found", "idLike", idLike, "regex", idLikeRegex)
				c.String(http.StatusNotFound, "ID LIKE %s not found", idLike)
				return
			}
		} else {
			slog.Warn("Parameter `id LIKE` not set")
			c.String(http.StatusBadRequest, "Parameter `id LIKE` not set")
			return
		}
	}
	if len(sessions) <= 0 {
		c.String(http.StatusNotFound, "No ids found for request")
		return
	}
	slog.Info("Requested sessions", "sessions", sessions)
	var responseBody strings.Builder
	for _, s := range sessions {
		ext := idToExt(s.Extension)
		cid := idToCid(s.Extension)
		slog.Info("Requested Extension", "extension", ext, "callerid", cid)
		switch c.Param("pstype") {
		case "ps_endpoint":
			endpoint := url.Values{}
			endpoint.Add("id", ext)
			endpoint.Add("auth", ext)
			endpoint.Add("aors", ext)
			for key, value := range SipEndpoint {
				endpoint.Add(key, value)
			}
			endpoint.Add("callerid", cid)
			slog.Info("Endpoint response", "response", endpoint.Encode())
			responseBody.WriteString(endpoint.Encode() + "\n")
		case "ps_auth":
			endpoint := url.Values{}
			endpoint.Add("id", ext)
			for key, value := range SipAuth {
				endpoint.Add(key, value)
			}
			endpoint.Add("username", ext)
			endpoint.Add("password", s.Password)
			responseBody.WriteString(endpoint.Encode() + "\n")
		case "ps_aor":
			endpoint := url.Values{}
			endpoint.Add("id", ext)
			for key, value := range SipAor {
				endpoint.Add(key, value)
			}
			endpoint.Add("mailboxes", ext+"@default")
			responseBody.WriteString(endpoint.Encode() + "\n")
		default:
			c.String(http.StatusBadRequest, "Request for %s unknown", c.Param("pstype"))
			return
		}
	}
	c.Data(http.StatusOK, "application/x-www-form-urlencoded", []byte(responseBody.String()))
	return
}
