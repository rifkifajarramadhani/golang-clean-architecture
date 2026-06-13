package password

import "testing"

func TestBcryptRoundTrip(t *testing.T) {
	hasher := Bcrypt{}
	hash, err := hasher.Hash("secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := hasher.Compare(hash, "secret"); err != nil {
		t.Fatal(err)
	}
	if err := hasher.Compare(hash, "wrong"); err == nil {
		t.Fatal("expected comparison failure")
	}
}
