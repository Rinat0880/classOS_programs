package monitor

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type BrowserMonitor struct {
	lastCheck map[string]time.Time
	callback  func(browser, url string)
	username  string
}

func NewBrowserMonitor(username string, callback func(browser, url string)) *BrowserMonitor {
	return &BrowserMonitor{
		lastCheck: make(map[string]time.Time),
		callback:  callback,
		username:  username,
	}
}

func (bm *BrowserMonitor) Start() {
	go bm.monitorLoop()
}

func (bm *BrowserMonitor) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		bm.checkBrowsers()
	}
}

func (bm *BrowserMonitor) checkBrowsers() {
	if bm.username == "" {
		return
	}

	cleanUsername := bm.cleanUsername(bm.username)
	
	bm.checkChrome(cleanUsername)
	bm.checkEdge(cleanUsername)
	bm.checkFirefox(cleanUsername)
}

func (bm *BrowserMonitor) cleanUsername(username string) string {
	if idx := strings.Index(username, "\\"); idx != -1 {
		return username[idx+1:]
	}
	return username
}

func (bm *BrowserMonitor) checkChrome(username string) {
	historyPath := fmt.Sprintf("C:\\Users\\%s\\AppData\\Local\\Google\\Chrome\\User Data\\Default\\History", username)
	bm.readChromeHistory(historyPath, "Chrome")
}

func (bm *BrowserMonitor) checkEdge(username string) {
	historyPath := fmt.Sprintf("C:\\Users\\%s\\AppData\\Local\\Microsoft\\Edge\\User Data\\Default\\History", username)
	bm.readChromeHistory(historyPath, "Edge")
}

func (bm *BrowserMonitor) checkFirefox(username string) {
	profilesPath := fmt.Sprintf("C:\\Users\\%s\\AppData\\Roaming\\Mozilla\\Firefox\\Profiles", username)
	
	entries, err := os.ReadDir(profilesPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".default-release") {
			historyPath := filepath.Join(profilesPath, entry.Name(), "places.sqlite")
			bm.readFirefoxHistory(historyPath)
			break
		}
	}
}

func (bm *BrowserMonitor) readChromeHistory(historyPath, browser string) {
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return
	}

	tempPath := historyPath + ".tmp"
	if err := bm.copyFile(historyPath, tempPath); err != nil {
		return
	}
	defer os.Remove(tempPath)

	db, err := sql.Open("sqlite3", tempPath)
	if err != nil {
		return
	}
	defer db.Close()

	lastCheck := bm.lastCheck[browser]
	if lastCheck.IsZero() {
		lastCheck = time.Now().Add(-5 * time.Minute)
	}

	query := `
		SELECT url, title, last_visit_time 
		FROM urls 
		WHERE last_visit_time > ? 
		ORDER BY last_visit_time DESC 
		LIMIT 50
	`

	chromeEpoch := time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)
	lastCheckMicro := lastCheck.Sub(chromeEpoch).Microseconds()

	rows, err := db.Query(query, lastCheckMicro)
	if err != nil {
		return
	}
	defer rows.Close()

	var maxTime time.Time
	count := 0

	for rows.Next() {
		var url, title string
		var visitTime int64

		if err := rows.Scan(&url, &title, &visitTime); err != nil {
			continue
		}

		visitTimestamp := chromeEpoch.Add(time.Duration(visitTime) * time.Microsecond)
		
		if visitTimestamp.After(lastCheck) {
			if visitTimestamp.After(maxTime) {
				maxTime = visitTimestamp
			}

			if bm.isImportantURL(url) {
				action := fmt.Sprintf("Visited: %s", url)
				if title != "" && len(title) < 100 {
					action = fmt.Sprintf("Visited: %s (%s)", url, title)
				}
				bm.callback(browser, action)
				count++
			}
		}
	}

	if !maxTime.IsZero() {
		bm.lastCheck[browser] = maxTime
	}

	if count > 0 {
		log.Printf("Logged %d new %s visits", count, browser)
	}
}

func (bm *BrowserMonitor) readFirefoxHistory(historyPath string) {
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return
	}

	tempPath := historyPath + ".tmp"
	if err := bm.copyFile(historyPath, tempPath); err != nil {
		return
	}
	defer os.Remove(tempPath)

	db, err := sql.Open("sqlite3", tempPath)
	if err != nil {
		return
	}
	defer db.Close()

	lastCheck := bm.lastCheck["Firefox"]
	if lastCheck.IsZero() {
		lastCheck = time.Now().Add(-5 * time.Minute)
	}

	query := `
		SELECT url, title, last_visit_date 
		FROM moz_places 
		WHERE last_visit_date > ? 
		ORDER BY last_visit_date DESC 
		LIMIT 50
	`

	lastCheckMicro := lastCheck.UnixMicro()

	rows, err := db.Query(query, lastCheckMicro)
	if err != nil {
		return
	}
	defer rows.Close()

	var maxTime time.Time
	count := 0

	for rows.Next() {
		var url, title string
		var visitTime int64

		if err := rows.Scan(&url, &title, &visitTime); err != nil {
			continue
		}

		visitTimestamp := time.UnixMicro(visitTime)
		
		if visitTimestamp.After(lastCheck) {
			if visitTimestamp.After(maxTime) {
				maxTime = visitTimestamp
			}

			if bm.isImportantURL(url) {
				action := fmt.Sprintf("Visited: %s", url)
				if title != "" && len(title) < 100 {
					action = fmt.Sprintf("Visited: %s (%s)", url, title)
				}
				bm.callback("Firefox", action)
				count++
			}
		}
	}

	if !maxTime.IsZero() {
		bm.lastCheck["Firefox"] = maxTime
	}

	if count > 0 {
		log.Printf("Logged %d new Firefox visits", count)
	}
}

func (bm *BrowserMonitor) isImportantURL(url string) bool {
	if strings.HasPrefix(url, "chrome://") || 
	   strings.HasPrefix(url, "edge://") ||
	   strings.HasPrefix(url, "about:") ||
	   strings.HasPrefix(url, "chrome-extension://") ||
	   strings.HasPrefix(url, "moz-extension://") {
		return false
	}

	if strings.Contains(url, "google.com/search") ||
	   strings.Contains(url, "bing.com/search") ||
	   strings.Contains(url, "youtube.com/watch") ||
	   strings.Contains(url, "github.com") ||
	   strings.Contains(url, "stackoverflow.com") ||
	   strings.Contains(url, "facebook.com") ||
	   strings.Contains(url, "twitter.com") ||
	   strings.Contains(url, "instagram.com") ||
	   strings.Contains(url, "reddit.com") ||
	   strings.Contains(url, "wikipedia.org") ||
	   len(url) < 200 {
		return true
	}

	return false
}

func (bm *BrowserMonitor) copyFile(src, dst string) error {
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

func (bm *BrowserMonitor) UpdateUsername(username string) {
	bm.username = username
}
