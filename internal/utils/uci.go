package utils

import (
	"log/slog"
	"os/exec"
	"strings"
)

func UciGet(key string) (string, error) {
	output, err := exec.Command("uci", "get", key).Output()
	if err != nil {
		slog.Error("uci_get error", "err", err)
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func UciCheck() error {
	return exec.Command("uci").Run()
}
