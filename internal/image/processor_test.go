package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

// makeFileHeader builds a *multipart.FileHeader carrying the given bytes, the
// same way an HTTP upload would.
func makeFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("images", filename)
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("writing form file failed: %v", err)
	}
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("ReadForm failed: %v", err)
	}
	t.Cleanup(func() { form.RemoveAll() })

	files := form.File["images"]
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	return files[0]
}

// pngBytes returns an encoded solid-color PNG.
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode failed: %v", err)
	}
	return buf.Bytes()
}

func newTestProcessor(t *testing.T) *Processor {
	t.Helper()
	return NewProcessor(t.TempDir(), 10*1024*1024, []string{".jpg", ".jpeg", ".png", ".gif"})
}

func TestProcessUploadHappyPath(t *testing.T) {
	p := newTestProcessor(t)
	fh := makeFileHeader(t, "photo.png", pngBytes(t, 800, 600))

	filename, thumbnail, err := p.ProcessUpload(fh)
	if err != nil {
		t.Fatalf("ProcessUpload failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(p.uploadDir, filename)); err != nil {
		t.Errorf("original file was not saved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.uploadDir, thumbnail)); err != nil {
		t.Errorf("thumbnail was not saved: %v", err)
	}

	// Thumbnail must fit within the configured bounds
	width, height, _, err := GetImageInfo(filepath.Join(p.uploadDir, thumbnail))
	if err != nil {
		t.Fatalf("GetImageInfo failed: %v", err)
	}
	if width > p.thumbnailSize || height > p.thumbnailSize {
		t.Errorf("thumbnail is %dx%d, want at most %dx%d", width, height, p.thumbnailSize, p.thumbnailSize)
	}
}

func TestProcessUploadRejectsDisallowedExtension(t *testing.T) {
	p := newTestProcessor(t)
	fh := makeFileHeader(t, "malware.exe", []byte("MZ..."))

	if _, _, err := p.ProcessUpload(fh); err == nil {
		t.Error("expected ProcessUpload to reject a .exe file")
	}
}

func TestProcessUploadRejectsFakeImageContent(t *testing.T) {
	p := newTestProcessor(t)
	// Right extension, but the bytes are not an image
	fh := makeFileHeader(t, "fake.png", []byte("<html>not an image</html>"))

	if _, _, err := p.ProcessUpload(fh); err == nil {
		t.Error("expected ProcessUpload to reject non-image content with an image extension")
	}
}

func TestProcessUploadRejectsOversizedFile(t *testing.T) {
	p := NewProcessor(t.TempDir(), 10, []string{".png"}) // 10-byte limit
	fh := makeFileHeader(t, "big.png", pngBytes(t, 50, 50))

	if _, _, err := p.ProcessUpload(fh); err == nil {
		t.Error("expected ProcessUpload to reject a file over the size limit")
	}
}

func TestDeleteImage(t *testing.T) {
	p := newTestProcessor(t)
	fh := makeFileHeader(t, "photo.png", pngBytes(t, 100, 100))

	filename, thumbnail, err := p.ProcessUpload(fh)
	if err != nil {
		t.Fatalf("ProcessUpload failed: %v", err)
	}

	if err := p.DeleteImage(filename); err != nil {
		t.Fatalf("DeleteImage failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(p.uploadDir, filename)); !os.IsNotExist(err) {
		t.Error("original file still exists after DeleteImage")
	}
	if _, err := os.Stat(filepath.Join(p.uploadDir, thumbnail)); !os.IsNotExist(err) {
		t.Error("thumbnail still exists after DeleteImage")
	}

	// Deleting a nonexistent image is not an error
	if err := p.DeleteImage("does-not-exist.png"); err != nil {
		t.Errorf("DeleteImage on missing file returned error: %v", err)
	}
}
