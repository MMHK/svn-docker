package config

import (
	"os"
	"strings"
)

// applyLegacyINI is a read-only compatibility layer for the v2
// data/config.ini. Environment settings always win; the file is only consulted
// for the three Subversion paths and is never written back.
//
// v2 naming is confusing: in config.ini SVNAuthFile points at the authz
// (access) file and SVNUserFile points at the htpasswd file, so they are
// mapped onto SVN_ACCESS_FILE and SVN_AUTH_FILE respectively.
func (c *Config) applyLegacyINI() error {
	if c.ConfigINI == "" {
		return nil
	}

	content, err := os.ReadFile(c.ConfigINI)
	if err != nil {
		// Missing or unreadable legacy file is not fatal: env remains the
		// authoritative source.
		return nil
	}

	values := parseINI(string(content))

	if !c.explicit.parent {
		if value := values["SVNParentPath"]; value != "" {
			c.SVNParentPath = value
		}
	}

	if !c.explicit.access {
		if value := values["SVNAuthFile"]; value != "" {
			c.SVNAccessFile = value
		}
	}

	if !c.explicit.auth {
		if value := values["SVNUserFile"]; value != "" {
			c.SVNAuthFile = value
		}
	}

	return nil
}

// parseINI is a deliberately small, lossless-enough reader for the flat
// key=value structure used by data/config.ini. Keys from every section share
// one namespace because the three legacy keys are unique in practice.
func parseINI(content string) map[string]string {
	values := make(map[string]string)

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" || value == "" {
			continue
		}

		// First occurrence wins so an operator override at the top of the file
		// is not clobbered by a later section.
		if _, exists := values[key]; !exists {
			values[key] = value
		}
	}

	return values
}
