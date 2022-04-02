package handlers

import (
	"fmt"
	"io"
	"net/http"
)

func formatSeconds(seconds int) string {
	var (
		sec  = seconds
		min  = 0
		hour = 0
	)

	if sec >= 60 {
		min = sec / 60
		sec = sec % 60
	}

	if min >= 60 {
		hour = min / 60
		min = min % 60
	}

	timeStr := fmt.Sprintf("%02d:%02d", min, sec)
	if hour > 0 {
		timeStr = fmt.Sprintf("%02d:%s", hour, timeStr)
	}

	return timeStr
}

func getFile(link string) ([]byte, error) {
	resp, err := http.Get(link)
	if err != nil {
		return nil, fmt.Errorf("get error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return io.ReadAll(resp.Body)
}
