package core

import (
	_ "embed"
	"strings"
)

//go:generate sh -c "git describe --tags --long --dirty='-devel' --match '[0-9]*.[0-9]*.[0-9]*' > version.txt"
//go:embed version.txt
var version string

// builtVersion can be set at build time, e.g. by goreleaser:
//
//	-ldflags "-X github.com/davidolrik/corto/internal/core.builtVersion=1.2.3"
var builtVersion string

var Version = func() string {
	if builtVersion != "" {
		return builtVersion
	}
	return strings.TrimSpace(version)
}()
