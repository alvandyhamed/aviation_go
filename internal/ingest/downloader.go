package ingest

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func downloadToTemp(url string) (string, error) {
	f, err := os.CreateTemp("", "dl-*.csv")

	if err != nil {
		return "", err
	}
	defer f.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad status: %s", resp.Status)

	}
	_, err = io.Copy(f, resp.Body)
	return f.Name(), err
}
