package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type SipUser struct {
	username string
	password string
	callerid string
}

// TODO: remove these temporary test userds when implementing actual user generation
var data = map[string]SipUser{
	"user0100": {
		username: "user0100",
		password: "1234",
		callerid: "\"websocket user 0100\" <2100>",
	},
	"user0101": {
		username: "user0101",
		password: "1234",
		callerid: "\"websocket user 0101\" <2101>",
	},
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
	fmt.Printf("Request for %s single?: \n%t", c.Param("pstype"), is_single)
	var ids []string
	if is_single {
		id, found := c.GetPostForm("id")
		if found {
			if _, ok := data[id]; ok {
				ids = append(ids, id)
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
			re, err := regexp.Compile(idLikeRegex)
			if err != nil {
				fmt.Println("Error compiling regex:", err)
				return
			} else {
				found := false
				for id := range data {
					if re.MatchString(id) {
						found = true
						ids = append(ids, id)
					}
				}
				if !found {
					log.Printf("ID LIKE %s converted to %s but not found", idLike, idLikeRegex)
				}
			}
		} else {
			log.Printf("Parameter `id LIKE` not set")
			c.String(http.StatusBadRequest, "Parameter `id LIKE` not set")
			return
		}
	}
	if len(ids) <= 0 {
		c.String(http.StatusNotFound, "No ids found for request")
		return
	}
	fmt.Println("Requested ids: ", ids)
	var responseBody strings.Builder
	for _, id := range ids {
		switch c.Param("pstype") {
		case "ps_endpoint":
			endpoint := url.Values{}
			endpoint.Add("id", id)
			endpoint.Add("auth", id)
			endpoint.Add("aors", id)
			for key, value := range SipEndpoint {
				endpoint.Add(key, value)
			}
			endpoint.Add("callerid", data[id].callerid)
			responseBody.WriteString(endpoint.Encode() + "\n")
		case "ps_auth":
			endpoint := url.Values{}
			endpoint.Add("id", id)
			for key, value := range SipAuth {
				endpoint.Add(key, value)
			}
			endpoint.Add("username", data[id].username)
			endpoint.Add("password", data[id].password)
			responseBody.WriteString(endpoint.Encode() + "\n")
		case "ps_aor":
			endpoint := url.Values{}
			endpoint.Add("id", id)
			for key, value := range SipAor {
				endpoint.Add(key, value)
			}
			endpoint.Add("mailboxes", id+"@default")
			responseBody.WriteString(endpoint.Encode() + "\n")
		default:
			c.String(http.StatusBadRequest, "Request for %s unknown", c.Param("pstype"))
			return
		}
	}
	c.String(http.StatusOK, responseBody.String())
	return
}
