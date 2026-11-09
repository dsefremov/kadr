package jobs

import (
	"testing"
	"time"
)

func TestRetryDelay(t *testing.T) {
	if retryDelay(1) != time.Minute {
		t.Fatal("delay 1")
	}
	if retryDelay(5) != 2*time.Hour {
		t.Fatal("delay 5")
	}
}
