// ABOUTME: Tests for path security validation and directory traversal prevention
// ABOUTME: Covers prefix bypass, system directory write blocking, symlink resolution

package permission

import (
	"testing"
)

func TestSecurePathValidator_PrefixBypass(t *testing.T) {
	t.Parallel()

	v, err := NewSecurePathValidator([]string{"/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"inside allowed dir", "/tmp/test.txt", false},
		{"exact allowed dir", "/tmp", false},
		{"prefix bypass", "/tmpevil/secret.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := v.ValidateReadPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReadPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateWritePath_SystemDirPrefixBypass(t *testing.T) {
	t.Parallel()

	// Allow "/" so we can test the system directory check specifically
	v, err := NewSecurePathValidator([]string{"/"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"blocked /etc/passwd", "/etc/passwd", true},
		{"blocked /usr/bin/foo", "/usr/bin/foo", true},
		{"allowed /etc-config/foo (no false positive)", "/etc-config/foo", false},
		{"allowed /usr-local/bin", "/usr-local/bin", false},
		{"blocked exact /etc", "/etc", true},
		{"blocked /bin/sh", "/bin/sh", true},
		{"allowed /binary/data", "/binary/data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := v.ValidateWritePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWritePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestCheckTraversalPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"normal path", "/home/user/file.txt", false},
		{"dot-dot slash", "../../../etc/passwd", true},
		{"null byte", "/tmp/file\x00.txt", true},
		{"url-encoded traversal", "..%2f..%2f", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkTraversalPatterns(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTraversalPatterns(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
