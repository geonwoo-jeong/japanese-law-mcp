package streamablehttp

import (
	"net/http"
	"strings"
)

// servePreflight は、Origin 照合後に SOT-DEL-015 の事前確認だけを行う。
func servePreflight(writer http.ResponseWriter, request *http.Request) {
	methods := request.Header.Values("Access-Control-Request-Method")
	if request.Header.Get("Origin") == "" ||
		len(methods) != 1 || methods[0] != http.MethodPost ||
		!preflightHeadersAllowed(request.Header) || hasSessionHeader(request.Header) {
		http.Error(writer, "CORS preflight の Origin、method または header が不正です", http.StatusBadRequest)
		return
	}

	writer.Header().Set("Access-Control-Allow-Origin", request.Header.Get("Origin"))
	writer.Header().Set("Access-Control-Allow-Methods", http.MethodPost)
	writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, MCP-Protocol-Version")
	writer.WriteHeader(http.StatusNoContent)
}

func preflightHeadersAllowed(header http.Header) bool {
	for _, value := range header.Values("Access-Control-Request-Headers") {
		for name := range strings.SplitSeq(value, ",") {
			switch http.CanonicalHeaderKey(strings.Trim(name, " \t")) {
			case "Accept", "Content-Type", "Mcp-Protocol-Version":
			default:
				return false
			}
		}
	}
	return true
}
