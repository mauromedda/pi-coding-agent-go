// ABOUTME: Smart image resize pipeline with format and quality fallback
// ABOUTME: Uses CatmullRom interpolation; falls back to JPEG if PNG exceeds size limit

package image

import (
	"bytes"
	"fmt"
	goimage "image"
	"image/jpeg"
	"image/png"

	// Register decoders for standard formats.
	_ "image/gif"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Resize scales image data to fit within maxDim (pixels) and maxBytes (file size).
// It returns the resized bytes, final dimensions, MIME type, and any error.
//
// Algorithm:
//  1. If already within limits, return as-is.
//  2. Decode, resize respecting aspect ratio with CatmullRom.
//  3. Encode as PNG; if too large, try JPEG at decreasing quality.
//  4. If still too large, scale down further at 0.75, 0.5, 0.35, 0.25.
func Resize(data []byte, maxDim, maxBytes int) ([]byte, Dimensions, string, error) {
	if len(data) == 0 {
		return nil, Dimensions{}, "", fmt.Errorf("empty image data")
	}

	dim, err := GetDimensions(data)
	if err != nil {
		return nil, Dimensions{}, "", fmt.Errorf("reading dimensions: %w", err)
	}

	// Already within both limits
	if dim.Width <= maxDim && dim.Height <= maxDim && len(data) <= maxBytes {
		mime := detectMIME(data)
		return data, dim, mime, nil
	}

	// Decode the full image
	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, Dimensions{}, "", fmt.Errorf("decoding image: %w", err)
	}

	// Calculate target dimensions preserving aspect ratio
	targetW, targetH := fitDimensions(dim.Width, dim.Height, maxDim)
	resized := resizeImage(img, targetW, targetH)

	// Try encoding: PNG first, then JPEG with decreasing quality
	out, mime, err := encodeWithFallback(resized, maxBytes)
	if err != nil {
		return nil, Dimensions{}, "", err
	}

	if len(out) <= maxBytes {
		return out, Dimensions{Width: targetW, Height: targetH}, mime, nil
	}

	// Scale down further
	for _, scale := range []float64{0.75, 0.5, 0.35, 0.25} {
		sw := int(float64(targetW) * scale)
		sh := int(float64(targetH) * scale)
		if sw < 1 {
			sw = 1
		}
		if sh < 1 {
			sh = 1
		}
		scaled := resizeImage(img, sw, sh)
		out, mime, err = encodeWithFallback(scaled, maxBytes)
		if err != nil {
			return nil, Dimensions{}, "", err
		}
		if len(out) <= maxBytes {
			return out, Dimensions{Width: sw, Height: sh}, mime, nil
		}
	}

	// Return smallest attempt even if over limit
	finalDim := Dimensions{
		Width:  int(float64(targetW) * 0.25),
		Height: int(float64(targetH) * 0.25),
	}
	return out, finalDim, mime, nil
}

// fitDimensions calculates new dimensions that fit within maxDim while preserving aspect ratio.
func fitDimensions(w, h, maxDim int) (int, int) {
	if w <= maxDim && h <= maxDim {
		return w, h
	}
	if w >= h {
		return maxDim, h * maxDim / w
	}
	return w * maxDim / h, maxDim
}

// resizeImage scales an image to the target dimensions using CatmullRom interpolation.
func resizeImage(src goimage.Image, w, h int) goimage.Image {
	dst := goimage.NewRGBA(goimage.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// encodeWithFallback tries PNG first, then JPEG at decreasing quality levels.
func encodeWithFallback(img goimage.Image, maxBytes int) ([]byte, string, error) {
	// Try PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, "", fmt.Errorf("encoding PNG: %w", err)
	}
	if buf.Len() <= maxBytes {
		return buf.Bytes(), "image/png", nil
	}

	// Try JPEG at decreasing quality
	for _, q := range []int{85, 70, 55, 40} {
		buf.Reset()
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, "", fmt.Errorf("encoding JPEG: %w", err)
		}
		if buf.Len() <= maxBytes {
			return buf.Bytes(), "image/jpeg", nil
		}
	}

	return buf.Bytes(), "image/jpeg", nil
}

// detectMIME returns a MIME type based on the magic bytes of image data.
func detectMIME(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return "image/png"
	}
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}
	if data[0] == 'G' && data[1] == 'I' && data[2] == 'F' {
		return "image/gif"
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return "application/octet-stream"
}
