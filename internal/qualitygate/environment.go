package qualitygate

import (
	"os"
	"strings"
)

var controlledGoEnvironment = []string{
	"GOWORK",
	"GOENV",
	"GOTOOLCHAIN",
	"GOPROXY",
	"GOSUMDB",
	"GOPRIVATE",
	"GONOPROXY",
	"GONOSUMDB",
	"GOINSECURE",
	"GOVCS",
	"GOFLAGS",
	"GOCACHE",
	"GOLANGCI_LINT_CACHE",
	"GOOS",
	"GOARCH",
	"GOEXPERIMENT",
	"GO111MODULE",
	"CGO_ENABLED",
}

type cachePaths struct {
	goBuild  string
	golangci string
}

func goEnvironment(
	ambient []string,
	network bool,
	goFlags string,
	caches cachePaths,
) []string {
	environment := make([]string, 0, len(ambient)+len(controlledGoEnvironment))
	for _, entry := range ambient {
		key, _, _ := strings.Cut(entry, "=")
		if isControlledGoVariable(key) {
			continue
		}
		environment = append(environment, entry)
	}

	proxy := "off"
	if network {
		proxy = "https://proxy.golang.org"
	}
	environment = append(environment,
		"GOWORK=off",
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"GOPROXY="+proxy,
		"GOSUMDB=sum.golang.org",
		"GOPRIVATE=",
		"GONOPROXY=",
		"GONOSUMDB=",
		"GOINSECURE=",
		"GOVCS=public:git,private:off",
		"GOFLAGS="+goFlags,
		"GOCACHE="+caches.goBuild,
		"GOLANGCI_LINT_CACHE="+caches.golangci,
	)
	return environment
}

func gitEnvironment(
	ambient []string,
	preserveIndex bool,
	preserveObjects bool,
	isolateGlobal bool,
) []string {
	environment := make([]string, 0, len(ambient)+4)
	var indexFile string
	var objectDirectory string
	var alternateObjectDirectories string
	for _, entry := range ambient {
		key, value, ok := strings.Cut(entry, "=")
		switch {
		case strings.EqualFold(key, "GIT_INDEX_FILE") && ok:
			indexFile = value
		case strings.EqualFold(key, "GIT_OBJECT_DIRECTORY") && ok:
			objectDirectory = value
		case strings.EqualFold(key, "GIT_ALTERNATE_OBJECT_DIRECTORIES") && ok:
			alternateObjectDirectories = value
		}
		if strings.HasPrefix(strings.ToUpper(key), "GIT_") {
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(environment,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
	)
	if isolateGlobal {
		environment = append(environment, "GIT_CONFIG_GLOBAL="+os.DevNull)
	}
	if preserveIndex && indexFile != "" {
		environment = append(environment, "GIT_INDEX_FILE="+indexFile)
	}
	if preserveObjects && objectDirectory != "" {
		environment = append(environment, "GIT_OBJECT_DIRECTORY="+objectDirectory)
	}
	if preserveObjects && alternateObjectDirectories != "" {
		environment = append(
			environment,
			"GIT_ALTERNATE_OBJECT_DIRECTORIES="+alternateObjectDirectories,
		)
	}
	return environment
}

func isControlledGoVariable(key string) bool {
	if strings.HasPrefix(strings.ToUpper(key), "GOLANGCI_LINT_") {
		return true
	}
	for _, controlled := range controlledGoEnvironment {
		if strings.EqualFold(key, controlled) {
			return true
		}
	}
	return false
}
