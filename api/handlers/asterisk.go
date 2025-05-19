package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

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
	return fmt.Sprintf("Emergency_%s%04d<2%d>", defaultUserPrefix, id, id)
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
				log.Printf("Failed to read file %s: %v", filePath, err)
				continue
			}
			var session sipSession
			err = json.Unmarshal(session_content, &session)
			if err != nil {
				log.Fatal("Error unmarshaling JSON:", err)
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
					log.Printf("Failed to read file %s: %v", filePath, err)
					continue
				}
				var session sipSession
				err = json.Unmarshal(session_content, &session)
				if err != nil {
					log.Fatal("Error unmarshaling JSON:", err)
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
	//TODO: unify logging a bit (log vs print)
	fmt.Printf("Request for %s single?: %t\n", c.Param("pstype"), is_single)
	var sessions []sipSession
	if is_single {
		id, found := c.GetPostForm("id")
		if found {
			session, err := getSession(id)
			if err == nil {
				sessions = append(sessions, *session)
			} else {
				log.Printf("ID %s not found", id)
				c.String(http.StatusNotFound, "ID %s not found", id)
				return
			}
		} else {
			log.Printf("Parameter `id` not set")
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
				log.Printf("ID LIKE %s converted to %s but not found", idLike, idLikeRegex)
				c.String(http.StatusNotFound, "ID LIKE %s not found", idLike)
				return
			}
		} else {
			log.Printf("Parameter `id LIKE` not set")
			c.String(http.StatusBadRequest, "Parameter `id LIKE` not set")
			return
		}
	}
	if len(sessions) <= 0 {
		c.String(http.StatusNotFound, "No ids found for request")
		return
	}
	fmt.Println("Requested sessions: ", sessions)
	var responseBody strings.Builder
	for _, s := range sessions {
		ext := idToExt(s.Extension)
		cid := idToCid(s.Extension)
		fmt.Printf("Requested Extension: %s with Callerid: %s\n", ext, cid)
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
			fmt.Printf("Edpoint response: %s", endpoint.Encode())
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
