package module

import (
	"math/rand"
	"os"
	"strconv"
	"time"
)

func newTempDir() (string, error) {
	tmpDir := os.TempDir() + "/go-providence-checker-" + strconv.Itoa(rand.Int())
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", err
	}
	return tmpDir, nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
