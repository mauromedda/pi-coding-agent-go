// ABOUTME: Tests for binary image dimension extraction from header bytes
// ABOUTME: Covers PNG, JPEG, GIF formats with real minimal test images

package image

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeGIF(t *testing.T, w, h int) []byte {
	t.Helper()
	palette := []color.Color{color.Black, color.White}
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestGetDimensions_PNG(t *testing.T) {
	data := makePNG(t, 320, 240)
	dim, err := GetDimensions(data)
	if err != nil {
		t.Fatal(err)
	}
	if dim.Width != 320 || dim.Height != 240 {
		t.Errorf("got %dx%d, want 320x240", dim.Width, dim.Height)
	}
}

func TestGetDimensions_JPEG(t *testing.T) {
	data := makeJPEG(t, 640, 480)
	dim, err := GetDimensions(data)
	if err != nil {
		t.Fatal(err)
	}
	if dim.Width != 640 || dim.Height != 480 {
		t.Errorf("got %dx%d, want 640x480", dim.Width, dim.Height)
	}
}

func TestGetDimensions_GIF(t *testing.T) {
	data := makeGIF(t, 100, 50)
	dim, err := GetDimensions(data)
	if err != nil {
		t.Fatal(err)
	}
	if dim.Width != 100 || dim.Height != 50 {
		t.Errorf("got %dx%d, want 100x50", dim.Width, dim.Height)
	}
}

func TestGetDimensions_PNGFromHeader(t *testing.T) {
	// Verify our fast-path header parsing for PNG works with known bytes.
	// PNG IHDR: bytes 16-19 = width (big-endian), bytes 20-23 = height (big-endian)
	header := make([]byte, 24)
	copy(header, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}) // PNG signature
	// IHDR chunk length (13 bytes) at offset 8
	binary.BigEndian.PutUint32(header[8:12], 13)
	// "IHDR" at offset 12
	copy(header[12:16], "IHDR")
	// Width at offset 16
	binary.BigEndian.PutUint32(header[16:20], 1024)
	// Height at offset 20
	binary.BigEndian.PutUint32(header[20:24], 768)

	dim, err := GetDimensions(header)
	if err != nil {
		t.Fatal(err)
	}
	if dim.Width != 1024 || dim.Height != 768 {
		t.Errorf("got %dx%d, want 1024x768", dim.Width, dim.Height)
	}
}

func TestGetDimensions_TooShort(t *testing.T) {
	_, err := GetDimensions([]byte{0x89, 'P', 'N', 'G'})
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

func TestGetDimensions_UnknownFormat(t *testing.T) {
	_, err := GetDimensions([]byte("not an image at all"))
	if err == nil {
		t.Error("expected error for unknown format")
	}
}
