package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Device    string `json:"device,omitempty"`
	Message   string `json:"message"`
}

type Logger struct {
	mu            sync.Mutex
	logDir        string
	retentionDays int
}

var instance *Logger

func Init(logDir string, retentionDays int) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	instance = &Logger{
		logDir:        logDir,
		retentionDays: retentionDays,
	}
	go instance.cleanupRoutine()
	return nil
}

func Info(device, message string) {
	if instance == nil {
		return
	}
	instance.write("INFO", device, message)
}

func Error(device, message string) {
	if instance == nil {
		return
	}
	instance.write("ERROR", device, message)
}

func (l *Logger) write(level, device, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Level:     level,
		Device:    device,
		Message:   message,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Println("Error marshaling log entry:", err)
		return
	}

	filename := filepath.Join(l.logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}

func (l *Logger) cleanupRoutine() {
	for {
		l.cleanup()
		time.Sleep(24 * time.Hour)
	}
}

func (l *Logger) cleanup() {
	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -l.retentionDays)
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		// Filename format: YYYY-MM-DD.log
		name := file.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}
		
		dateStr := strings.TrimSuffix(name, ".log")
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if date.Before(cutoff) {
			os.Remove(filepath.Join(l.logDir, name))
		}
	}
}

// GetLogs returns logs, optionally filtered by device.
// It reads from the most recent log files up to a certain limit.
func GetLogs(deviceFilter string, limit int) ([]LogEntry, error) {
	if instance == nil {
		return nil, nil
	}
	
	instance.mu.Lock()
	defer instance.mu.Unlock()

	var logs []LogEntry
	
	files, err := os.ReadDir(instance.logDir)
	if err != nil {
		return nil, err
	}

	// Sort files by name (date) descending
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})

	count := 0
	for _, file := range files {
		if count >= limit {
			break
		}
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		f, err := os.Open(filepath.Join(instance.logDir, file.Name()))
		if err != nil {
			continue
		}
		
		// Read lines from file
		// Since we want most recent first, and file is appended, we should read all and reverse, 
		// or read from end. For simplicity, read all and filter.
		var fileLogs []LogEntry
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var entry LogEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
				if deviceFilter == "" || entry.Device == deviceFilter {
					fileLogs = append(fileLogs, entry)
				}
			}
		}
		f.Close()

		// Reverse fileLogs to get newest first
		for i := len(fileLogs) - 1; i >= 0; i-- {
			logs = append(logs, fileLogs[i])
			count++
			if count >= limit {
				break
			}
		}
	}

	return logs, nil
}
