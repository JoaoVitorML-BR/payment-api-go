package paymentmercadopago

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type SellerTokenStore interface {
	Save(userID string, token OAuthToken) error
	Load(userID string) (OAuthToken, error)
}

func (s *EncryptedFileTokenStore) Load(userID string) (OAuthToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, err := os.ReadFile(s.path)
	if err != nil {
		return OAuthToken{}, fmt.Errorf("read encrypted OAuth credentials: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return OAuthToken{}, fmt.Errorf("decode encrypted OAuth credentials: %w", err)
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return OAuthToken{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(data) < gcm.NonceSize() {
		return OAuthToken{}, errors.New("encrypted OAuth credentials are invalid")
	}
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return OAuthToken{}, errors.New("encrypted OAuth credentials could not be decrypted")
	}
	var stored struct {
		UserID string     `json:"user_id"`
		Token  OAuthToken `json:"token"`
	}
	if err := json.Unmarshal(plain, &stored); err != nil || stored.UserID != userID || stored.Token.UserID != userID || stored.Token.AccessToken == "" {
		return OAuthToken{}, errors.New("stored OAuth credentials do not match seller")
	}
	return stored.Token, nil
}

type EncryptedFileTokenStore struct {
	path string
	key  []byte
	mu   sync.Mutex
}

func NewEncryptedFileTokenStore(path, encodedKey string) (*EncryptedFileTokenStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("OAuth token file path is required")
	}
	key, err := decodeEncryptionKey(encodedKey)
	if err != nil {
		return nil, err
	}
	return &EncryptedFileTokenStore{path: path, key: key}, nil
}

func (s *EncryptedFileTokenStore) Save(userID string, token OAuthToken) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(token.AccessToken) == "" || strings.TrimSpace(token.RefreshToken) == "" {
		return errors.New("seller OAuth credentials are incomplete")
	}

	payload, err := json.Marshal(struct {
		UserID string     `json:"user_id"`
		Token  OAuthToken `json:"token"`
	}{UserID: userID, Token: token})
	if err != nil {
		return fmt.Errorf("encode seller OAuth credentials: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return fmt.Errorf("create OAuth encryption cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create OAuth encryption mode: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate OAuth encryption nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, payload, nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	s.mu.Lock()
	defer s.mu.Unlock()
	if directory := filepath.Dir(s.path); directory != "." {
		if err := os.MkdirAll(directory, 0700); err != nil {
			return fmt.Errorf("create encrypted OAuth credentials directory: %w", err)
		}
	}
	if err := os.WriteFile(s.path, []byte(encoded+"\n"), 0600); err != nil {
		return fmt.Errorf("write encrypted OAuth credentials: %w", err)
	}
	return nil
}

func decodeEncryptionKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(key) != 32 {
		return nil, errors.New("OAuth encryption key must be a base64-encoded 32-byte key")
	}
	return key, nil
}
