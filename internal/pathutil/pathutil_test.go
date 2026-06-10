package pathutil

import (
	"runtime"
	"testing"
)

func TestNormalizeWindows_MSYSPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"msys /c/", "/c/Users/test", "C:\\Users\\test"},
		{"msys /d/", "/d/AI/project", "D:\\AI\\project"},
		{"msys uppercase", "/C/Users/test", "C:\\Users\\test"},
		{"msys deep path", "/c/Users/59263/AppData/Roaming/reasonix/projects", "C:\\Users\\59263\\AppData\\Roaming\\reasonix\\projects"},
		{"already windows", "C:\\Users\\test", "C:\\Users\\test"},
		{"already windows D:", "D:\\AI\\project", "D:\\AI\\project"},
		{"pipe style /c|/", "/c|/Users/test", "C:\\Users\\test"},
		{"pipe style /d|/", "/d|/AI/project", "D:\\AI\\project"},
		{"not msys - too short", "/c", "/c"},
		{"not msys - no letter after slash", "/123/Users", "/123/Users"},
		{"not msys - multi letter", "/cde/Users", "/cde/Users"},
		{"relative path", "some/relative/path", "some/relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWindows(tt.in)
			if got != tt.want {
				t.Errorf("NormalizeWindows(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeWindows_NonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows only")
	}
	got := NormalizeWindows("/c/Users/test")
	if got != "/c/Users/test" {
		t.Errorf("on non-Windows, should pass through unchanged, got %q", got)
	}
}

func TestAbs_ReturnsAbsolute(t *testing.T) {
	got := Abs(".")
	if len(got) == 0 {
		t.Fatal("Abs returned empty string")
	}
	if runtime.GOOS == "windows" {
		if len(got) < 2 || got[1] != ':' {
			t.Errorf("Windows Abs should have drive letter, got %q", got)
		}
	}
}

func TestAbs_MSYSPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	got := Abs("/c/Users/test")
	if len(got) < 2 || got[1] != ':' {
		t.Errorf("MSYS path should be normalized to Windows path, got %q", got)
	}
}
