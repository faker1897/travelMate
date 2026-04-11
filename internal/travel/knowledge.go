package travel

import (
	"path/filepath"
	"strings"
)

func IsLegacyNonTravelDoc(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	legacyNames := []string{
		"告警处理手册.md",
		"告警处理手册.markdown",
		"oncall.md",
		"aiops.md",
	}
	for _, name := range legacyNames {
		if base == strings.ToLower(name) {
			return true
		}
	}
	return false
}
