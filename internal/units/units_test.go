package units

import "testing"

func TestDecimalQuota(t *testing.T) {
	if DefaultQuotaBytes != 1_000_000_000 {
		t.Fatalf("expected decimal 1 GB, got %d", DefaultQuotaBytes)
	}
	if PhotoMaxBytes != 50_000_000 {
		t.Fatal("photo max")
	}
	if VideoMaxBytes != 200_000_000 {
		t.Fatal("video max")
	}
	if VideoMaxDurationMs != 60*60*1000 {
		t.Fatal("video max duration")
	}
}
