package hanreipdf

import (
	"errors"
	"net/http"
)

var errUnsafeRedirect = errors.New("裁判所 PDF の redirect が許可範囲外です")

func newProductionHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableCompression = true
	transport.ProxyConnectHeader = nil
	return &http.Client{
		Transport:     transport,
		CheckRedirect: courtsPDFRedirectPolicy,
	}
}

func courtsPDFRedirectPolicy(request *http.Request, via []*http.Request) error {
	if request == nil || request.Method != http.MethodGet ||
		request.URL == nil || !isAllowedDocumentURL(request.URL.String()) ||
		len(via) >= 3 {
		return errUnsafeRedirect
	}
	for _, previous := range via {
		if previous == nil || previous.Method != http.MethodGet ||
			previous.URL == nil || !isAllowedDocumentURL(previous.URL.String()) {
			return errUnsafeRedirect
		}
	}
	request.Header.Del("Authorization")
	request.Header.Del("Cookie")
	request.Header.Del("Proxy-Authorization")
	request.Header.Del("Referer")
	return nil
}
