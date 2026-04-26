package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
)

const hookErrorFile = "last-hook-error.txt"

func hookErrorPath() string {
	cfg, err := config.Load()
	if err == nil {
		return filepath.Join(cfg.DataDir, hookErrorFile)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".devpulse", hookErrorFile)
}

func recordHookError(context string, err error) {
	if err == nil {
		return
	}
	path := hookErrorPath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	msg := fmt.Sprintf("%s  %s: %v\n", time.Now().Format(time.RFC3339), context, err)
	_ = os.WriteFile(path, []byte(msg), 0644)
}

func clearHookError() {
	path := hookErrorPath()
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

func readHookError() string {
	path := hookErrorPath()
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
