package riqu

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// QTransport wraps a reusable HTTP transport.
type QTransport struct {
	httpTransport *http.Transport
}
// NewTransport creates a transport using environment proxies and system CAs by default.
func NewTransport(options *TTransportOptions) (*QTransport, error) {
	// DEFAULTS - Clone the default transport to preserve connection and TLS settings.
	defaultTransport, isHTTPTransport := http.DefaultTransport.(*http.Transport)

	if !isHTTPTransport || defaultTransport == nil {
		return nil, fmt.Errorf("riqu: default HTTP transport cannot be cloned")
	}
	transport := defaultTransport.Clone()

	if options == nil {
		return &QTransport{httpTransport: transport}, nil
	}
	// PROXY - Validate and apply an explicit HTTP or HTTPS proxy.
	if options.ProxyURL != "" {
		proxyURL, err := url.Parse(options.ProxyURL)

		if err != nil {
			return nil, fmt.Errorf("riqu: invalid proxy URL: %w", err)
		}
		if proxyURL.Hostname() == "" || (proxyURL.Scheme != "http" && proxyURL.Scheme != "https") {
			return nil, fmt.Errorf("riqu: proxy URL must include an http or https scheme and a host")
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	// TRUST - Add certificates from the supplied PEM file to the system CA pool.
	if options.CAFile != "" {
		certPEM, err := os.ReadFile(options.CAFile)

		if err != nil {
			return nil, fmt.Errorf("riqu: read CA file: %w", err)
		}
		roots, err := x509.SystemCertPool()

		if err != nil {
			return nil, fmt.Errorf("riqu: load system CA certificates: %w", err)
		}
		if !roots.AppendCertsFromPEM(certPEM) {
			return nil, fmt.Errorf("riqu: no valid CA certificates found in %q", options.CAFile)
		}
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.RootCAs = roots
	}
	return &QTransport{httpTransport: transport}, nil
}
// CloseIdleConnections closes idle connections owned by the transport.
func (this *QTransport) CloseIdleConnections() {
	if this == nil || this.httpTransport == nil {
		return
	}
	this.httpTransport.CloseIdleConnections()
}
