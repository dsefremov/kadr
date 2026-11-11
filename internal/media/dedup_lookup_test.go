package media

import "testing"

func TestTrashRetentionDays(t *testing.T) {
	if TrashRetention.Hours()/24 != 30 {
		t.Fatalf("retention: %v", TrashRetention)
	}
}
