package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// getCurrentLogName returns a filename for a bot or event log
// relevant to the current date. Invalid inputs return an empty
// string.
func getCurrentLogName(logType LogType, sID string) string {
	switch logType {
	case bot:
		return "seb_bot_" + time.Now().Format("20060102") + ".log"
	case event:
		return "seb_event-" + sID + "_" + time.Now().Format("20060102") + ".log"
	default:
		return ""
	}
}

// New generates a new log file to write to.
//
// If the log file to generate is a bot log, sID is unused,
// and may be left empty.
func New(logType LogType, sID string) (*os.File, error) {
	subdirs := []string{}
	if logType == event {
		subdirs = append(subdirs, "events")
	} else {
		subdirs = append(subdirs, "bot")
	}
	path, _ := GetLogFilepath(getCurrentLogName(logType, sID), subdirs)

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return nil, fmt.Errorf("could not open or create directory: %s", err.Error())
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("could not open or create log file: %s", err.Error())
	}

	return file, nil
}

// Close checks the size of the given file and deletes it
// if it is empty.
func Close(file *os.File) error {
	if file != nil {
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("could not get file info for log: %s", err.Error())
		}
		if fileInfo.Size() == 0 {
			path, _ := GetLogFilepath(getCurrentLogName(bot, ""), []string{})

			if err := os.Remove(path); err != nil {
				return fmt.Errorf("could not remove log file: %s", err.Error())
			}
		} else if err := file.Close(); err != nil {
			panic(err)
		}
	}

	return nil
}
