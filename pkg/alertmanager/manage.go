package alertmanager

import (
	"fmt"
	"io"
	"net/http"
)

func Reload(url string) (*http.Response, error) {
	resp, err := http.Post(
		fmt.Sprintf("%s/-/reload", url),
		"application/x-www-form-urlencoded",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed reload alertmanager: %s", err)
	}

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, fmt.Errorf("response body read failed: %s", err)
		}

		return resp, fmt.Errorf("failed alertmanager reload with status code \"%d\" and body \"%s\"", resp.StatusCode, body)
	}

	return resp, nil
}
