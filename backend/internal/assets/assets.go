package assets

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"dash/backend/internal/store"
)

var allowedMIMEs = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

type Service struct {
	dir      string
	baseURL  string
	maxBytes int64
	store    *store.Store
}

func New(dir, publicBaseURL string, maxBytes int64, st *store.Store) *Service {
	return &Service{
		dir:      dir,
		baseURL:  strings.TrimRight(publicBaseURL, "/"),
		maxBytes: maxBytes,
		store:    st,
	}
}

func (s *Service) SaveIcon(r *http.Request, header *multipart.FileHeader) (store.AssetFile, error) {
	if header.Size > s.maxBytes {
		return store.AssetFile{}, ErrTooLarge
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return store.AssetFile{}, err
	}

	file, err := header.Open()
	if err != nil {
		return store.AssetFile{}, err
	}
	defer file.Close()

	limited := io.LimitReader(file, s.maxBytes+1)
	sniff := make([]byte, 512)
	n, err := limited.Read(sniff)
	if err != nil && !errors.Is(err, io.EOF) {
		return store.AssetFile{}, err
	}
	mimeType := http.DetectContentType(sniff[:n])
	if strings.EqualFold(filepath.Ext(header.Filename), ".svg") {
		if !looksLikeSVG(sniff[:n]) {
			return store.AssetFile{}, ErrUnsupportedMedia
		}
		mimeType = "image/svg+xml"
	}
	ext, ok := allowedMIMEs[mimeType]
	if !ok {
		return store.AssetFile{}, ErrUnsupportedMedia
	}

	storedName := uuid.NewString() + ext
	target := filepath.Join(s.dir, storedName)
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return store.AssetFile{}, err
	}
	defer out.Close()

	if _, err := out.Write(sniff[:n]); err != nil {
		return store.AssetFile{}, err
	}
	written, err := io.Copy(out, limited)
	if err != nil {
		return store.AssetFile{}, err
	}
	size := int64(n) + written
	if size > s.maxBytes {
		_ = os.Remove(target)
		return store.AssetFile{}, ErrTooLarge
	}

	return s.store.CreateAsset(r.Context(), store.AssetFile{
		OriginalName: filepath.Base(header.Filename),
		StoredName:   storedName,
		MimeType:     mimeType,
		SizeBytes:    size,
		PublicPath:   fmt.Sprintf("/uploads/icons/%s", storedName),
	})
}

func looksLikeSVG(prefix []byte) bool {
	trimmed := bytes.TrimSpace(prefix)
	return bytes.HasPrefix(trimmed, []byte("<svg")) || bytes.HasPrefix(trimmed, []byte("<?xml"))
}

type assetError string

func (e assetError) Error() string { return string(e) }

const (
	ErrTooLarge         assetError = "upload too large"
	ErrUnsupportedMedia assetError = "unsupported media type"
)
