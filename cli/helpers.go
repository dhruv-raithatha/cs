package cli

import "os"

// ExpandHome replaces a leading "~/" with the user's home directory.
func ExpandHome(path string) string {
	return expandHome(path)
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}
