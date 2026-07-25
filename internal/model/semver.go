package model

import "strings"

func isSemVer(value string) bool {
	version := value
	if separator := strings.IndexByte(version, '+'); separator >= 0 {
		build := version[separator+1:]
		version = version[:separator]
		if !validIdentifiers(build, false) {
			return false
		}
	}

	if separator := strings.IndexByte(version, '-'); separator >= 0 {
		prerelease := version[separator+1:]
		version = version[:separator]
		if !validIdentifiers(prerelease, true) {
			return false
		}
	}

	core := strings.Split(version, ".")
	if len(core) != 3 {
		return false
	}
	for _, part := range core {
		if !isNumericIdentifier(part) || hasLeadingZero(part) {
			return false
		}
	}
	return true
}

func validIdentifiers(value string, rejectNumericLeadingZero bool) bool {
	if value == "" {
		return false
	}
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" || !isASCIIIdentifier(identifier) {
			return false
		}
		if rejectNumericLeadingZero &&
			isNumericIdentifier(identifier) &&
			hasLeadingZero(identifier) {
			return false
		}
	}
	return true
}

func isASCIIIdentifier(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'A' || character > 'Z') &&
			(character < 'a' || character > 'z') &&
			character != '-' {
			return false
		}
	}
	return true
}

func isNumericIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func hasLeadingZero(value string) bool {
	return len(value) > 1 && value[0] == '0'
}
