/*
 * MIT License
 *
 * Copyright (c) 2026 Roman Bielyi
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package helper

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
)

// GetTLSConfig returns TLS configuration for mTLS connection.
// Run examples from project root: go run ./examples/code/go/...
func GetTLSConfig() (*tls.Config, error) {
	clientCert, err := tls.LoadX509KeyPair("scripts/certs/client.crt", "scripts/certs/client.key")
	if err != nil {
		log.Fatalf("Failed to read client certificate key pair: %s", err)
	}

	caCertPEM, err := os.ReadFile("scripts/certs/ca.crt")
	if err != nil {
		log.Fatalf("Failed to read CA certificate file: %s", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		log.Fatal("failed to append CA certificate")
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
		ServerName:   "xpropagator-server",
	}

	return tlsCfg, nil
}
