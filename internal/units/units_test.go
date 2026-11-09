package units

import "testing"

func TestDecimalQuota(t *testing.T) {
	if DefaultQuotaBytes != 5_000_000_000 {
		t.Fatalf("expected decimal 5 GB, got %d", DefaultQuotaBytes)
	}
	if PhotoMaxBytes != 50_000_000 {
		t.Fatal("photo max")
	}
}
