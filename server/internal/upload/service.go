package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

var allowedExts = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true,
}

// Service handles image file uploads.
type Service struct {
	uploadDir   string
	urlPrefix   string
	maxFileSize int64
}

// NewService constructs an upload service.
//   - uploadDir: local directory root for stored files (e.g. "./uploads")
//   - urlPrefix: URL prefix returned in responses (e.g. "/uploads")
//   - maxFileSize: max allowed bytes (e.g. 5242880 = 5 MB)
func NewService(uploadDir, urlPrefix string, maxFileSize int64) *Service {
	return &Service{
		uploadDir:   uploadDir,
		urlPrefix:   urlPrefix,
		maxFileSize: maxFileSize,
	}
}

// Upload validates and saves the uploaded image file.
// Returns the public URL path (urlPrefix/yyyy-MM-dd/hexuuid.ext) on success.
// Matches Java UploadService.upload() error messages and path scheme exactly.
func (s *Service) Upload(file *multipart.FileHeader) (string, error) {
	if file == nil || file.Size == 0 {
		return "", httpx.Biz("文件不能为空")
	}
	if file.Size > s.maxFileSize {
		return "", httpx.Biz("文件不能超过 5MB")
	}

	origin := file.Filename
	if origin == "" || !strings.Contains(origin, ".") {
		return "", httpx.Biz("文件名非法")
	}
	ext := strings.ToLower(origin[strings.LastIndex(origin, ".")+1:])
	if !allowedExts[ext] {
		return "", httpx.Biz("仅支持 jpg/jpeg/png/gif/webp")
	}

	// Dated subdirectory: yyyy-MM-dd (matches Java LocalDate.now().toString())
	dateDir := time.Now().Format("2006-01-02")
	dir := filepath.Join(s.uploadDir, dateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", httpx.Biz("创建目录失败")
	}

	// 32-char hex filename (matches Java UUID.randomUUID().toString().replace("-",""))
	filename := randomHex32() + "." + ext
	dest := filepath.Join(dir, filename)

	if err := saveFile(file, dest); err != nil {
		return "", httpx.Biz("保存文件失败")
	}

	// URL = urlPrefix + "/" + dateDir + "/" + filename
	// e.g. /uploads/2026-07-23/abc123...jpg
	url := s.urlPrefix + "/" + dateDir + "/" + filename
	return url, nil
}

// randomHex32 generates 16 random bytes and returns them as a 32-char hex string,
// matching Java's UUID.randomUUID().toString().replace("-","") (also 32 hex chars).
func randomHex32() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback: use nanosecond timestamp (should never happen in practice)
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func saveFile(fh *multipart.FileHeader, dest string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	return saveFromReader(src, dest)
}

// saveFromReader copies src into dest, cleaning up the partial file on error.
// Separated from saveFile to allow unit-testing the cleanup path without a real
// multipart.FileHeader.
func saveFromReader(src io.Reader, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	copyErr := func() error {
		_, err := io.Copy(out, src)
		return err
	}()
	closeErr := out.Close()

	if copyErr != nil {
		_ = os.Remove(dest) // remove partial file on write error
		return copyErr
	}
	return closeErr
}
