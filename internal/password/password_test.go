package password

import "testing"

func TestHashAndCompare(t *testing.T) {
	if err := Validate("short"); err == nil {
		t.Fatal("expected validation error")
	}
	hash, err := Hash("longenoughpw")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := Compare(hash, "longenoughpw")
	if err != nil || !ok {
		t.Fatalf("compare=%v err=%v", ok, err)
	}
	ok, _ = Compare(hash, "wrongpasswordx")
	if ok {
		t.Fatal("should not match")
	}
}
