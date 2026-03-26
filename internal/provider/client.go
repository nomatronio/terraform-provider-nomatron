package provider

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func buildHTTPClient(data NomatronProviderModel) (*http.Client, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if !data.TlsCaCert.IsNull() && data.TlsCaCert.ValueString() != "" {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(data.TlsCaCert.ValueString())) {
			return nil, fmt.Errorf("failed to parse tls_ca_cert PEM")
		}
		tlsConfig.RootCAs = pool
	}

	hasClientCert := !data.TlsCert.IsNull() && data.TlsCert.ValueString() != ""
	hasClientKey := !data.TlsKey.IsNull() && data.TlsKey.ValueString() != ""

	if hasClientCert || hasClientKey {
		if !hasClientCert || !hasClientKey {
			return nil, fmt.Errorf("tls_cert and tls_key must both be set")
		}

		cert, err := tls.X509KeyPair(
			[]byte(data.TlsCert.ValueString()),
			[]byte(data.TlsKey.ValueString()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate/key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()

	if tlsConfig.RootCAs != nil || len(tlsConfig.Certificates) > 0 {
		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

func normalizeAddress(raw string, useTLS bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	if !strings.Contains(raw, "://") {
		scheme := "http"
		if useTLS {
			scheme = "https"
		}
		raw = scheme + "://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if u.Path == "" || u.Path == "/" {
		u.Path = "/api/v1"
	}

	return strings.TrimRight(u.String(), "/")
}
