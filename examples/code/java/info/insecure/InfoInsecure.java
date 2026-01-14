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

package insecure;

import api.v1.Info.InfoResponse;
import api.v1.PropagatorGrpc;
import com.google.protobuf.Empty;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

import java.time.Instant;

/**
 * Example demonstrating how to call the api.v1.Propagator.Info method
 * using an insecure (plaintext) connection.
 *
 * WARNING: This should only be used for local development/testing.
 * In production, always use mTLS (see InfoSecure.java).
 */
public class InfoInsecure {

    private static final String SERVER_HOST = "localhost";
    private static final int SERVER_PORT = 50051;

    public static void main(String[] args) {
        ManagedChannel channel = null;
        try {
            // Build insecure channel (plaintext, no TLS)
            channel = ManagedChannelBuilder.forAddress(SERVER_HOST, SERVER_PORT)
                    .usePlaintext()
                    .build();

            // Create blocking stub for synchronous calls
            PropagatorGrpc.PropagatorBlockingStub stub = PropagatorGrpc.newBlockingStub(channel);

            // Call the Info method
            InfoResponse response = stub.info(Empty.getDefaultInstance());

            // Print the response
            System.out.println("=== api.v1.Propagator.Info Response ===");
            System.out.println("Name:            " + response.getName());
            System.out.println("Version:         " + response.getVersion());
            System.out.println("Commit:          " + response.getCommit());
            System.out.println("Build Date:      " + response.getBuildDate());
            System.out.println("AstroStdLibInfo: " + response.getAstroStdLibInfo());
            System.out.println("Sgp4LibInfo:     " + response.getSgp4LibInfo());

            // Convert protobuf Timestamp to Java Instant
            Instant timestamp = Instant.ofEpochSecond(
                    response.getTimestamp().getSeconds(),
                    response.getTimestamp().getNanos()
            );
            System.out.println("Timestamp:       " + timestamp);

        } catch (Exception e) {
            System.err.println("Failed to call api.v1.Propagator.Info: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        } finally {
            if (channel != null) {
                channel.shutdownNow();
            }
        }
    }
}
