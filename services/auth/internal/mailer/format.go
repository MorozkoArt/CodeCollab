package mailer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
)

func messageID(fromAddr string) (string, error) {
	i := strings.LastIndex(fromAddr, "@")
	if i < 0 || i == len(fromAddr)-1 {
		return "", fmt.Errorf("message id: no domain in %q", fromAddr)
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("message id: %w", err)
	}
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(buf), fromAddr[i+1:]), nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(enum.DateTimeUserFormat) + " UTC"
}

func minutes(d time.Duration) string {
	n := int(d.Minutes())
	word := "минут"
	switch {
	case n%100 >= 11 && n%100 <= 14:
	case n%10 == 1:
		word = "минуту"
	case n%10 >= 2 && n%10 <= 4:
		word = "минуты"
	}
	return fmt.Sprintf("%d %s", n, word)
}
