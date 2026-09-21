package paymentmercadopago

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptedFileTokenStoreDoesNotWritePlaintext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "seller-token.enc")
	store, err := NewEncryptedFileTokenStore(path, base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save("seller-1", OAuthToken{AccessToken: "access-secret", RefreshToken: "refresh-secret", UserID: "seller-1"}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) == "" || string(contents) == "access-secret" || string(contents) == "refresh-secret" {
		t.Fatalf("token file is empty or plaintext: %q", contents)
	}
}
