// Package pathutil provides cross-platform path resolution utilities,
// hardened for Windows environments including Git Bash (MSYS2).
//
// On Windows, Git Bash and MSYS2 tools may set environment variables or
// pass paths in MSYS-style format (e.g. /c/Users/...) instead of proper
// Windows paths (e.g. C:\Users\...). This package normalizes such paths
// so downstream code can rely on consistent, absolute paths.
package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Abs returns the absolute form of p, falling back to a cleaned p on error.
// On Windows, it additionally normalizes MSYS-style paths (e.g. /c/Users/...)
// to proper Windows paths (e.g. C:\Users\...) so that subsequent file
// operations create directories in the correct location.
func Abs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}
	if runtime.GOOS == "windows" {
		abs = NormalizeWindows(abs)
	}
	return abs
}

// NormalizeWindows converts MSYS-style paths to proper Windows paths.
// On non-Windows platforms, it returns p unchanged.
//
// MSYS2 uses paths like /c/Users/... where "c" is the drive letter.
// A native Windows Go binary sees this as a relative path (no drive letter),
// which causes os.MkdirAll to create directories relative to the current
// drive root or even relative to CWD in some edge cases.
//
// Patterns recognized:
//   - /<letter>/...  -> <LETTER>:\...  (e.g. /c/Users -> C:\Users)
//   - /<letter>|/... -> <LETTER>:\...  (e.g. /d|/AI -> D:\AI)
func NormalizeWindows(p string) string {
	if runtime.GOOS != "windows" {
		return p
	}

	// Already a proper Windows absolute path (e.g. C:\Users).
	if len(p) >= 2 && p[1] == ':' {
		return p
	}

	// MSYS-style: /<drive-letter>/rest...
	// The drive letter must be a single ASCII letter.
	backslash := byte(92) // ASCII for backslash
	if len(p) >= 3 && p[0] == '/' && isASCIILetter(p[1]) && (p[2] == '/' || p[2] == backslash) {
		drive := strings.ToUpper(string(p[1]))
		rest := filepath.FromSlash(p[2:]) // normalize separators
		return drive + ":" + rest
	}

	// MSYS-style with pipe: /<letter>|/rest...
	if len(p) >= 4 && p[0] == '/' && isASCIILetter(p[1]) && p[2] == '|' {
		drive := strings.ToUpper(string(p[1]))
		rest := filepath.FromSlash(p[3:])
		return drive + ":" + rest
	}

	return p
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
