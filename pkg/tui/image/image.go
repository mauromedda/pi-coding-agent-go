// ABOUTME: Image utility for handling clipboard images and base64 encoding
// ABOUTME: Supports iTerm2 and Kitty image protocols

package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Protocol represents the image display protocol.
type Protocol int

const (
	ProtocolITerm2 Protocol = iota
	ProtocolKitty
)

// String returns the protocol name.
func (p Protocol) String() string {
	switch p {
	case ProtocolITerm2:
		return "iterm2"
	case ProtocolKitty:
		return "kitty"
	default:
		return "unknown"
	}
}

// DetectProtocol detects the image protocol from the terminal environment.
func DetectProtocol() Protocol {
	// Check for iTerm2
	if os.Getenv("ITERM_SESSION_ID") != "" {
		return ProtocolITerm2
	}

	// Check for Kitty
	if os.Getenv("KITTY_PID") != "" {
		return ProtocolKitty
	}

	// Check TERM_PROGRAM
	term := os.Getenv("TERM_PROGRAM")
	if term == "iTerm.app" {
		return ProtocolITerm2
	}

	// Default to iTerm2 on macOS
	if runtime.GOOS == "darwin" {
		return ProtocolITerm2
	}

	return ProtocolITerm2
}

// ClipboardImage reads an image from the clipboard.
// On macOS, uses `osascript` to get the clipboard image data.
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

// clipboardImageMacOS reads clipboard image using AppleScript.
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
		// Try alternative method using pngpaste if available
		cmd = exec.Command("pngpaste", "-")
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("reading clipboard image: %w", err)
		}
	}

	// If we got raw data (not base64), return it
	if len(output) > 0 {
		return output, nil
	}

	return nil, fmt.Errorf("no image data in clipboard")
}

// clipboardImageLinux reads clipboard image using xclip or xsel.
func clipboardImageLinux() ([]byte, error) {
	// Try xclip first
	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}

	// Try xsel
	cmd = exec.Command("xsel", "-b", "--input")
	output, err = cmd.Output()
	if err == nil {
		return output, nil
	}

	return nil, fmt.Errorf("reading clipboard image: %w", err)
}

// clipboardImageWindows reads clipboard image using PowerShell.
func clipboardImageWindows() ([]byte, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms;
$bmp = [System.Windows.Forms.Clipboard]::GetImage();
if ($bmp -ne $null) {
    $ms = New-Object System.IO.MemoryStream;
    $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png);
    $ms.ToArray();
}`;

	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("reading clipboard image: %w", err)
	}

	return output, nil
}

// EncodeImageBase64 encodes image data to base64.
func EncodeImageBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// ImageRef creates an image reference for display.
// Uses iTerm2 protocol: \033]1337;File=base64:<data>\a
func ImageRef(data []byte) string {
	encoded := EncodeImageBase64(data)
	protocol := DetectProtocol()

	switch protocol {
	case ProtocolKitty:
		// Kitty protocol: \033_Gfile=<base64>\033\
		return fmt.Sprintf("\033_Gfile=%s\033\\", encoded)
	default:
		// iTerm2 protocol: \033]1337;File=base64:<data>\a
		return fmt.Sprintf("\033]1337;File=base64:%s\a", encoded)
	}
}

// ImagePlaceholder creates a placeholder text for images.
func ImagePlaceholder(data []byte) string {
	size := len(data)
	return fmt.Sprintf("[Image: %d bytes]", size)
}

// PreviewImage creates an image preview (inline display).
func PreviewImage(data []byte) string {
	encoded := EncodeImageBase64(data)
	protocol := DetectProtocol()

	switch protocol {
	case ProtocolKitty:
		// kitty inline: \033[Gkey=value;...]\033\\
		return fmt.Sprintf("\033[Ginline=1;size=%d;filename=image.png;base64=%s\033\\",
			len(data), encoded)
	default:
		// iTerm2 inline: \033]1337;File=inline=1;<data>\a
		return fmt.Sprintf("\033]1337;File=inline=1;base64=%s\a", encoded)
	}
}

// EscapeSequence returns the escape sequence for image display.
func EscapeSequence(data []byte) string {
	encoded := EncodeImageBase64(data)

	// Build the image reference
	var buf bytes.Buffer
	buf.WriteString("\x1b]1337;File=base64:")
	buf.WriteString(encoded)
	buf.WriteString("\x1b\\")
	return buf.String()
}
