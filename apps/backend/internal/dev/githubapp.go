package dev

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// GitHub App auth: RS256 JWT (app id + PEM private key) exchanged for a
// short-lived installation token. Tokens cache ~50min; GitHub expires them
// after 60.
type appTokenCache struct {
	mu     sync.Mutex
	token  string
	expiry time.Time
	key    string // appId+key fingerprint so key rotation refetches
}

var ghAppCache = &appTokenCache{}

// appJWT builds the RS256 JWT GitHub requires for app-level calls.
func appJWT(appID, pemKey string) (string, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return "", fmt.Errorf("invalid PEM private key")
	}
	key, err := x509ParsePKCS1or8(block.Bytes)
	if err != nil {
		return "", err
	}
	now := time.Now()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appID,
	})
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	h := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func x509ParsePKCS1or8(der []byte) (*rsa.PrivateKey, error) {
	if k, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return rk, nil
}

// InstallationToken returns a cached installation access token.
func (g GitHub) InstallationToken(ctx context.Context, host, appID, pemKey, installationID string) (string, error) {
	fp := appID + "|" + pemKey[:min(len(pemKey), 64)] + "|" + installationID
	ghAppCache.mu.Lock()
	if ghAppCache.token != "" && ghAppCache.key == fp && time.Now().Before(ghAppCache.expiry) {
		t := ghAppCache.token
		ghAppCache.mu.Unlock()
		return t, nil
	}
	ghAppCache.mu.Unlock()

	jwt, err := appJWT(appID, pemKey)
	if err != nil {
		return "", err
	}
	u := fmt.Sprintf("%s/app/installations/%s/access_tokens", g.apiBase(host), installationID)
	var resp struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := ghDo(ctx, http.MethodPost, u, jwt, map[string]any{}, &resp); err != nil {
		return "", err
	}
	exp := time.Now().Add(50 * time.Minute)
	if t, err := time.Parse(time.RFC3339, resp.ExpiresAt); err == nil {
		exp = t.Add(-10 * time.Minute)
	}
	ghAppCache.mu.Lock()
	ghAppCache.token, ghAppCache.expiry, ghAppCache.key = resp.Token, exp, fp
	ghAppCache.mu.Unlock()
	return resp.Token, nil
}

// AppInstallations lists installations visible to the app (for picking one).
func (g GitHub) AppInstallations(ctx context.Context, host, appID, pemKey string) ([]struct {
	ID         int64  `json:"id"`
	Account    string `json:"account"`
	Repository string `json:"repository_selection"`
}, error) {
	jwt, err := appJWT(appID, pemKey)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Installations []struct {
			ID      int64 `json:"id"`
			Account struct {
				Login string `json:"login"`
			} `json:"account"`
			RepositorySelection string `json:"repository_selection"`
		} `json:"installations"`
	}
	if err := ghDo(ctx, http.MethodGet, g.apiBase(host)+"/app/installations", jwt, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]struct {
		ID         int64  `json:"id"`
		Account    string `json:"account"`
		Repository string `json:"repository_selection"`
	}, 0, len(resp.Installations))
	for _, i := range resp.Installations {
		out = append(out, struct {
			ID         int64  `json:"id"`
			Account    string `json:"account"`
			Repository string `json:"repository_selection"`
		}{i.ID, i.Account.Login, i.RepositorySelection})
	}
	return out, nil
}

// AppDiscover lists repos an installation can access.
func (g GitHub) AppDiscover(ctx context.Context, host, appID, pemKey, installationID string) ([]RepoRef, error) {
	tok, err := g.InstallationToken(ctx, host, appID, pemKey, installationID)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/installation/repositories?per_page=100", g.apiBase(host))
	var resp struct {
		Repositories []struct {
			FullName string `json:"full_name"`
			Name     string `json:"name"`
			Private  bool   `json:"private"`
		} `json:"repositories"`
	}
	if err := ghDo(ctx, http.MethodGet, u, tok, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]RepoRef, 0, len(resp.Repositories))
	for _, r := range resp.Repositories {
		out = append(out, RepoRef{Path: r.FullName, Name: r.Name, Host: host, Private: r.Private})
	}
	return out, nil
}
