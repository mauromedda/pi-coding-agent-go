// ABOUTME: Cross-platform clipboard image reading (macOS, Linux, Windows)
// ABOUTME: Utility functions for base64 encoding and text placeholders

package image

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
)

// ClipboardImage reads an image from the system clipboard.
// Returns raw PNG/image bytes.
func ClipboardImage() ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return clipboardImageMacOS()
	case "linux":
		return clipboardImageLinux()
	case "windows":
		return clipboardImageWindows()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func clipboardImageMacOS() ([]byte, error) {
	script := `tell application "System Events"
    set the clipboard_type to type of (the clipboard)
    if the clipboard_type is «class PNGf» or the clipboard_type is «class JPEG» or the clipboard_type is «class TIFF» then
        get the clipboard as «class PNGf»
    else
        error "No image in clipboard"
    end if
end tell`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("pngpaste", "-")
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("reading clipboard image: %w", err)
		}
	}

	if len(output) > 0 {
		return output, nil
	}
	return nil, fmt.Errorf("no image data in clipboard")
}

func clipboardImageLinux() ([]byte, error) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}

	cmd = exec.Command("xsel", "-b", "--input")
	output, err = cmd.Output()
	if err == nil {
		return output, nil
	}

	return nil, fmt.Errorf("reading clipboard image: %w", err)
}

func clipboardImageWindows() ([]byte, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms;
$bmp = [System.Windows.Forms.Clipboard]::GetImage();
if ($bmp -ne $null) {
    $ms = New-Object System.IO.MemoryStream;
    $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png);
    $ms.ToArray();
}`

	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("reading clipboard image: %w", err)
	}
	return output, nil
}

// EncodeImageBase64 encodes raw bytes to a base64 string.
func EncodeImageBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// ImagePlaceholder returns a text placeholder for an image.
func ImagePlaceholder(data []byte) string {
	return fmt.Sprintf("[Image: %d bytes]", len(data))
}
