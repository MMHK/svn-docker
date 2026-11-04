package view

import "time"

// PathInfo reports the state of one configured path without exposing file
// contents.
type PathInfo struct {
	Path     string `json:"path" example:"/var/svnlocal"`
	Exists   bool   `json:"exists" example:"true"`
	IsDir    bool   `json:"isDir" example:"true"`
	Readable bool   `json:"readable" example:"true"`
	Writable bool   `json:"writable" example:"true"`
}

// SystemInfo is the GET /system/info payload. It is the only endpoint allowed
// to reveal absolute paths.
type SystemInfo struct {
	Version       string    `json:"version" example:"3.0.0-dev"`
	GoVersion     string    `json:"goVersion" example:"go1.24.0"`
	Platform      string    `json:"platform" example:"linux/amd64"`
	SVNParentPath PathInfo  `json:"svnParentPath"`
	SVNAuthFile   PathInfo  `json:"svnAuthFile"`
	SVNAccessFile PathInfo  `json:"svnAccessFile"`
	DocsEnabled   bool      `json:"docsEnabled" example:"true"`
	DefaultLocale string    `json:"defaultLocale" example:"zh_CN"`
	ServerTime    time.Time `json:"serverTime"`
}

// ReadinessCheck is one named readiness probe. Message must never contain an
// absolute path.
type ReadinessCheck struct {
	Name    string `json:"name" example:"svnParentPath"`
	OK      bool   `json:"ok" example:"true"`
	Message string `json:"message,omitempty" example:"missing"`
}

// Readiness is returned by GET /readyz.
type Readiness struct {
	Status string           `json:"status" example:"ok"`
	Checks []ReadinessCheck `json:"checks"`
}
