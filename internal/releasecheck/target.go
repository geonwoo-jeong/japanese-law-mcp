package releasecheck

import "path/filepath"

type releaseTarget struct {
	goos        string
	goarch      string
	format      string
	binaryName  string
	archiveName string
}

func releaseTargets(version string) []releaseTarget {
	return []releaseTarget{
		newReleaseTarget(version, "darwin", "amd64", "tar.gz"),
		newReleaseTarget(version, "darwin", "arm64", "tar.gz"),
		newReleaseTarget(version, "windows", "amd64", "zip"),
		newReleaseTarget(version, "windows", "arm64", "zip"),
	}
}

func newReleaseTarget(version, goos, goarch, format string) releaseTarget {
	binaryName := projectName
	if goos == "windows" {
		binaryName += ".exe"
	}
	return releaseTarget{
		goos:        goos,
		goarch:      goarch,
		format:      format,
		binaryName:  binaryName,
		archiveName: projectName + "_" + version + "_" + goos + "_" + goarch + "." + format,
	}
}

func findReleaseTarget(goos, goarch, version string) (releaseTarget, bool) {
	for _, target := range releaseTargets(version) {
		if target.goos == goos && target.goarch == goarch {
			return target, true
		}
	}
	return releaseTarget{}, false
}

func (target releaseTarget) archivePath(dist string) string {
	return filepath.Join(dist, target.archiveName)
}
