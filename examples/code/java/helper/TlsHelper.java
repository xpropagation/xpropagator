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

package helper;

import io.grpc.netty.shaded.io.grpc.netty.GrpcSslContexts;
import io.grpc.netty.shaded.io.netty.handler.ssl.SslContext;

import javax.net.ssl.SSLException;
import java.io.File;

/**
 * Helper class for setting up mTLS (mutual TLS) for gRPC connections.
 * Run examples from project root: ./gradlew run...
 */
public class TlsHelper {

    private static final String CA_CERT_PATH = "scripts/certs/ca.crt";
    private static final String CLIENT_CERT_PATH = "scripts/certs/client.crt";
    private static final String CLIENT_KEY_PATH = "scripts/certs/client.key";

    /**
     * Builds an SslContext configured for mTLS with the XPropagator server.
     *
     * @return Configured SslContext for use with NettyChannelBuilder
     * @throws SSLException if there's an error building the SSL context
     */
    public static SslContext buildSslContext() throws SSLException {
        File caCert = new File(CA_CERT_PATH);
        File clientCert = new File(CLIENT_CERT_PATH);
        File clientKey = new File(CLIENT_KEY_PATH);

        validateFile(caCert, "CA certificate");
        validateFile(clientCert, "Client certificate");
        validateFile(clientKey, "Client private key");

        return GrpcSslContexts.forClient()
                .trustManager(caCert)
                .keyManager(clientCert, clientKey)
                .build();
    }

    private static void validateFile(File file, String description) {
        if (!file.exists()) {
            throw new IllegalArgumentException(
                    description + " not found: " + file.getAbsolutePath());
        }
        if (!file.canRead()) {
            throw new IllegalArgumentException(
                    description + " is not readable: " + file.getAbsolutePath());
        }
    }
}
