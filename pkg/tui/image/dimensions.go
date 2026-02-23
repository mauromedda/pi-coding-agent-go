// ABOUTME: Fast image dimension extraction from header bytes
// ABOUTME: Parses PNG, JPEG, GIF, and WebP without full image decode

package image

import (
	"encoding/binary"
	"fmt"
)

// Dimensions holds the width and height of an image.
type Dimensions struct {
	Width  int
	Height int
}

// GetDimensions extracts width and height from image header bytes.
// Supports PNG, JPEG, GIF, and WebP. Returns an error for unrecognized formats
// or truncated data.
func GetDimensions(data []byte) (Dimensions, error) {
	if len(data) < 8 {
		return Dimensions{}, fmt.Errorf("data too short (%d bytes)", len(data))
	}

	// PNG: 8-byte signature 0x89 P N G \r \n 0x1A \n
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return parsePNGDimensions(data)
	}

	// JPEG: starts with 0xFF 0xD8
	if data[0] == 0xFF && data[1] == 0xD8 {
		return parseJPEGDimensions(data)
	}

	// GIF: starts with "GIF87a" or "GIF89a"
	if data[0] == 'G' && data[1] == 'I' && data[2] == 'F' {
		return parseGIFDimensions(data)
	}

	// WebP: starts with "RIFF" ... "WEBP"
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return parseWebPDimensions(data)
	}

	return Dimensions{}, fmt.Errorf("unrecognized image format")
}

// parsePNGDimensions reads width/height from the IHDR chunk.
// Bytes 16-19: width (big-endian uint32), bytes 20-23: height (big-endian uint32).
func parsePNGDimensions(data []byte) (Dimensions, error) {
	if len(data) < 24 {
		return Dimensions{}, fmt.Errorf("PNG data too short for IHDR")
	}
	w := int(binary.BigEndian.Uint32(data[16:20]))
	h := int(binary.BigEndian.Uint32(data[20:24]))
	return Dimensions{Width: w, Height: h}, nil
}

// parseJPEGDimensions scans for SOF markers (0xFFC0-0xFFC2) to find dimensions.
func parseJPEGDimensions(data []byte) (Dimensions, error) {
	i := 2
	for i < len(data)-1 {
		if data[i] != 0xFF {
			i++
			continue
		}
		marker := data[i+1]

		// SOF0, SOF1, SOF2 markers
		if marker >= 0xC0 && marker <= 0xC2 {
			if i+9 >= len(data) {
				return Dimensions{}, fmt.Errorf("JPEG SOF truncated")
			}
			h := int(binary.BigEndian.Uint16(data[i+5 : i+7]))
			w := int(binary.BigEndian.Uint16(data[i+7 : i+9]))
			return Dimensions{Width: w, Height: h}, nil
		}

		// Skip non-SOF markers by reading their length
		if i+3 >= len(data) {
			break
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if segLen < 2 {
			break // Invalid segment length
		}
		i += 2 + segLen
		if i >= len(data) {
			break
		}
	}
	return Dimensions{}, fmt.Errorf("JPEG SOF marker not found")
}

// parseGIFDimensions reads width/height from the logical screen descriptor.
// Bytes 6-7: width (little-endian uint16), bytes 8-9: height (little-endian uint16).
func parseGIFDimensions(data []byte) (Dimensions, error) {
	if len(data) < 10 {
		return Dimensions{}, fmt.Errorf("GIF data too short for header")
	}
	w := int(binary.LittleEndian.Uint16(data[6:8]))
	h := int(binary.LittleEndian.Uint16(data[8:10]))
	return Dimensions{Width: w, Height: h}, nil
}

// parseWebPDimensions handles VP8, VP8L, and VP8X chunk formats.
func parseWebPDimensions(data []byte) (Dimensions, error) {
	if len(data) < 16 {
		return Dimensions{}, fmt.Errorf("WebP data too short")
	}

	chunk := string(data[12:16])
	switch chunk {
	case "VP8 ":
		// Lossy: frame header at offset 26; width at 26, height at 28 (little-endian uint16, masked)
		if len(data) < 30 {
			return Dimensions{}, fmt.Errorf("WebP VP8 data too short")
		}
		w := int(binary.LittleEndian.Uint16(data[26:28])) & 0x3FFF
		h := int(binary.LittleEndian.Uint16(data[28:30])) & 0x3FFF
		return Dimensions{Width: w, Height: h}, nil

	case "VP8L":
		// Lossless: signature byte at offset 21, then 4 bytes of packed w/h
		if len(data) < 25 {
			return Dimensions{}, fmt.Errorf("WebP VP8L data too short")
		}
		bits := binary.LittleEndian.Uint32(data[21:25])
		w := int(bits&0x3FFF) + 1
		h := int((bits>>14)&0x3FFF) + 1
		return Dimensions{Width: w, Height: h}, nil

	case "VP8X":
		// Extended: canvas size at offset 24 (3 bytes width, 3 bytes height, +1 each)
		if len(data) < 30 {
			return Dimensions{}, fmt.Errorf("WebP VP8X data too short")
		}
		w := int(data[24]) | int(data[25])<<8 | int(data[26])<<16 + 1
		h := int(data[27]) | int(data[28])<<8 | int(data[29])<<16 + 1
		return Dimensions{Width: w, Height: h}, nil

	default:
		return Dimensions{}, fmt.Errorf("unknown WebP chunk: %s", chunk)
	}
}
