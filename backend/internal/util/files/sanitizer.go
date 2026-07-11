package files_utils

// SanitizeFilename replaces characters that are invalid or problematic in filenames
// across different operating systems (Windows, Linux, macOS) and storage systems
// (local filesystem, S3, FTP, SFTP, NAS, rclone, Azure Blob).
//
// The following characters are replaced:
//   - Space (' ') -> underscore ('_')
//   - Forward slash ('/') -> hyphen ('-')
//   - Backslash ('\') -> hyphen ('-')
//   - Colon (':') -> hyphen ('-')
//   - Asterisk ('*') -> hyphen ('-')
//   - Question mark ('?') -> hyphen ('-')
//   - Double quote ('"') -> hyphen ('-')
//   - Less than ('<') -> hyphen ('-')
//   - Greater than ('>') -> hyphen ('-')
//   - Pipe ('|') -> hyphen ('-')
//
// This ensures filenames work correctly on:
//   - Windows (strict filename rules)
//   - Unix/Linux/macOS (forward slashes are path separators)
//   - All cloud storage providers (S3, Azure Blob)
//   - Network storage (FTP, SFTP, NAS, rclone)
func SanitizeFilename(name string) string {
	replacer := map[rune]rune{
		' ':  '_',
		'/':  '-',
		'\\': '-',
		':':  '-',
		'*':  '-',
		'?':  '-',
		'"':  '-',
		'<':  '-',
		'>':  '-',
		'|':  '-',
	}

	result := make([]rune, 0, len(name))
	for _, char := range name {
		if replacement, exists := replacer[char]; exists {
			result = append(result, replacement)
		} else {
			result = append(result, char)
		}
	}

	// A bare "." or ".." would survive as a real path segment and let a name used
	// as a folder traverse out of its storage root, so neutralize those exactly.
	switch sanitized := string(result); sanitized {
	case ".":
		return "_"
	case "..":
		return "__"
	default:
		return sanitized
	}
}
