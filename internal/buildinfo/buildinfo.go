package buildinfo

var version = "dev"

// Version は、ビルド時に設定されたバージョンを返す。
func Version() string {
	return version
}
