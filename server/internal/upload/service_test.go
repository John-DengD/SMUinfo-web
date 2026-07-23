package upload

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// makeFileHeader creates a *multipart.FileHeader with the given filename and content.
// Avoids needing a real HTTP request in unit tests.
func makeFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("createFormFile: %v", err)
	}
	if _, err := io.Copy(fw, bytes.NewReader(content)); err != nil {
		t.Fatalf("copy content: %v", err)
	}
	w.Close()

	r := multipart.NewReader(&buf, w.Boundary())
	form, err := r.ReadForm(10 << 20)
	if err != nil {
		t.Fatalf("readForm: %v", err)
	}
	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("no file in parsed form")
	}
	return files[0]
}

func bizMsg(err error) string {
	var be httpx.BizError
	if errors.As(err, &be) {
		return be.Msg
	}
	return ""
}

func TestUploadEmptyFileReturnsError(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 5<<20)
	fh := makeFileHeader(t, "test.jpg", []byte{})
	_, err := svc.Upload(fh)
	if bizMsg(err) != "文件不能为空" {
		t.Fatalf("expected 文件不能为空, got %v", err)
	}
}

func TestUploadTooLargeReturnsError(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 10) // max 10 bytes
	fh := makeFileHeader(t, "test.jpg", bytes.Repeat([]byte("x"), 20))
	_, err := svc.Upload(fh)
	if bizMsg(err) != "文件不能超过 5MB" {
		t.Fatalf("expected 文件不能超过 5MB, got %v", err)
	}
}

func TestUploadInvalidExtensionReturnsError(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 5<<20)
	fh := makeFileHeader(t, "malware.exe", []byte("data"))
	_, err := svc.Upload(fh)
	if bizMsg(err) != "仅支持 jpg/jpeg/png/gif/webp" {
		t.Fatalf("expected extension error, got %v", err)
	}
}

func TestUploadNoExtensionReturnsError(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 5<<20)
	fh := makeFileHeader(t, "noextension", []byte("data"))
	_, err := svc.Upload(fh)
	if bizMsg(err) != "文件名非法" {
		t.Fatalf("expected 文件名非法, got %v", err)
	}
}

func TestUploadSuccessURLPrefixAndDateDir(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	fh := makeFileHeader(t, "photo.jpg", []byte{0xff, 0xd8, 0xff}) // fake JPEG header
	url, err := svc.Upload(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// URL must start with /uploads/<today>/
	today := time.Now().Format("2006-01-02")
	expectedPrefix := "/uploads/" + today + "/"
	if !strings.HasPrefix(url, expectedPrefix) {
		t.Fatalf("URL %q does not start with %q", url, expectedPrefix)
	}
	// URL must end with .jpg
	if !strings.HasSuffix(url, ".jpg") {
		t.Fatalf("URL %q does not end with .jpg", url)
	}
}

func TestUploadSuccessFileExistsOnDisk(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	fh := makeFileHeader(t, "photo.png", []byte{0x89, 0x50, 0x4e, 0x47}) // fake PNG header
	url, err := svc.Upload(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reconstruct disk path from URL: /uploads/2026-07-23/abc.png → dir/2026-07-23/abc.png
	// URL format: /uploads/<dateDir>/<filename>
	parts := strings.SplitN(strings.TrimPrefix(url, "/uploads/"), "/", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected URL format: %q", url)
	}
	diskPath := filepath.Join(dir, parts[0], parts[1])
	if _, statErr := os.Stat(diskPath); os.IsNotExist(statErr) {
		t.Fatalf("file not found on disk at %q (URL=%q)", diskPath, url)
	}
}

func TestUploadHexFilenameIs32Chars(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	fh := makeFileHeader(t, "img.png", []byte{0x89, 0x50, 0x4e, 0x47})
	url, err := svc.Upload(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// base = filename without extension
	base := strings.TrimSuffix(filepath.Base(url), ".png")
	if len(base) != 32 {
		t.Fatalf("expected 32-char hex filename, got %d chars: %q", len(base), base)
	}
	// verify it's all hex chars
	for _, ch := range base {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Fatalf("filename contains non-hex char %q in %q", string(ch), base)
		}
	}
}

func TestAllowedExtensionsAllAccepted(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	for _, ext := range []string{"jpg", "jpeg", "png", "gif", "webp"} {
		fh := makeFileHeader(t, "file."+ext, []byte("img-data"))
		if _, err := svc.Upload(fh); err != nil {
			t.Fatalf("extension %q should be accepted, got error: %v", ext, err)
		}
	}
}

func TestExtensionIsCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	fh := makeFileHeader(t, "PHOTO.JPG", []byte("img-data"))
	if _, err := svc.Upload(fh); err != nil {
		t.Fatalf("uppercase .JPG should be accepted, got error: %v", err)
	}
}

// errReader returns an error after reading `n` bytes successfully.
type errReader struct {
	data []byte
	pos  int
	fail int // fail after this many bytes
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos >= r.fail {
		return 0, fmt.Errorf("simulated read error")
	}
	n := copy(p, r.data[r.pos:])
	if r.pos+n > r.fail {
		n = r.fail - r.pos
	}
	r.pos += n
	if r.pos >= r.fail {
		return n, fmt.Errorf("simulated read error")
	}
	return n, nil
}

// TestSaveFromReaderCleansUpPartialFileOnError verifies that saveFromReader
// removes the partially-written destination file when io.Copy fails.
func TestSaveFromReaderCleansUpPartialFileOnError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "partial.jpg")

	// Write 4 bytes of real data then error
	src := &errReader{data: []byte("abcdefgh"), fail: 4}
	err := saveFromReader(src, dest)
	if err == nil {
		t.Fatal("expected error from saveFromReader, got nil")
	}

	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatal("expected partial file to be removed after write error, but it still exists")
	}
}
