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

package single_propagate.insecure;

import api.v1.Common.EphemerisData;
import api.v1.Common.Satellite;
import api.v1.Prop.PropRequest;
import api.v1.Prop.PropResponse;
import api.v1.Prop.PropTask;
import api.v1.PropagatorGrpc;
import com.google.protobuf.Timestamp;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;

/**
 * Example demonstrating single satellite propagation using UTC time type
 * with an insecure (plaintext) connection.
 *
 * UTC time uses a protobuf Timestamp for human-readable date/time specification.
 *
 * WARNING: This should only be used for local development/testing.
 * In production, always use mTLS (see SinglePropagateUtcSecure.java).
 */
public class SinglePropagateUtcInsecure {

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

            long startTime = System.currentTimeMillis();

            // Build satellite TLE data
            Satellite satellite = Satellite.newBuilder()
                    .setNoradId(65271)
                    .setName("X-37B Orbital Test Vehicle 8 (OTV 8)")
                    .setTleLn1("1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07")
                    .setTleLn2("2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05")
                    .build();

            // Create UTC timestamp for propagation time: 2025-12-18 00:00:00 UTC
            ZonedDateTime utcTime = ZonedDateTime.of(2025, 12, 18, 0, 0, 0, 0, ZoneOffset.UTC);
            Instant instant = utcTime.toInstant();
            Timestamp timestamp = Timestamp.newBuilder()
                    .setSeconds(instant.getEpochSecond())
                    .setNanos(instant.getNano())
                    .build();

            // Build propagation task with UTC time
            PropTask task = PropTask.newBuilder()
                    .setSat(satellite)
                    .setTimeUtc(timestamp)
                    .build();

            // Build the propagation request (no TimeType needed for UTC)
            PropRequest request = PropRequest.newBuilder()
                    .setReqId(1)
                    .setTask(task)
                    .build();

            // Execute propagation
            PropResponse response = stub.prop(request);

            long elapsedMs = System.currentTimeMillis() - startTime;

            // Print results
            EphemerisData result = response.getResult();
            System.out.println("=== api.v1.Propagator.Prop Response ===");
            System.out.println("Request ID:  " + response.getReqId());
            System.out.println("Prop Time:   " + utcTime);
            System.out.println("Time (DS50): " + result.getDs50Time());
            System.out.println("Time (MSE):  " + result.getMseTime());
            System.out.println("Position (km):");
            System.out.printf("  X: %.6f%n", result.getX());
            System.out.printf("  Y: %.6f%n", result.getY());
            System.out.printf("  Z: %.6f%n", result.getZ());
            System.out.println("Velocity (km/s):");
            System.out.printf("  VX: %.6f%n", result.getVx());
            System.out.printf("  VY: %.6f%n", result.getVy());
            System.out.printf("  VZ: %.6f%n", result.getVz());
            System.out.println();
            System.out.println("Time elapsed: " + elapsedMs + " ms");

        } catch (Exception e) {
            System.err.println("Failed to call api.v1.Propagator.Prop: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        } finally {
            if (channel != null) {
                channel.shutdownNow();
            }
        }
    }
}
