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

package single_propagate.secure;

import api.v1.Common.EphemerisData;
import api.v1.Common.Satellite;
import api.v1.Prop.PropRequest;
import api.v1.Prop.PropResponse;
import api.v1.Prop.PropTask;
import api.v1.Prop.TimeType;
import api.v1.PropagatorGrpc;
import helper.TlsHelper;
import io.grpc.ManagedChannel;
import io.grpc.netty.shaded.io.grpc.netty.NettyChannelBuilder;
import io.grpc.netty.shaded.io.netty.handler.ssl.SslContext;

/**
 * Example demonstrating single satellite propagation using DS50 time type
 * with mTLS (mutual TLS) for secure communication.
 *
 * DS50 time is the number of days since 1950 Jan 0.0 UTC (a common astrodynamics epoch).
 */
public class SinglePropagateDs50Secure {

    private static final String SERVER_HOST = "localhost";
    private static final int SERVER_PORT = 50051;

    public static void main(String[] args) {
        ManagedChannel channel = null;
        try {
            // Build mTLS-enabled channel
            SslContext sslContext = TlsHelper.buildSslContext();

            channel = NettyChannelBuilder.forAddress(SERVER_HOST, SERVER_PORT)
                    .sslContext(sslContext)
                    .overrideAuthority("xpropagator-server")
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

            // Build propagation task with DS50 time
            PropTask task = PropTask.newBuilder()
                    .setSat(satellite)
                    .setTime(27744.5)  // DS50 time value
                    .build();

            // Build the propagation request
            PropRequest request = PropRequest.newBuilder()
                    .setReqId(1)
                    .setTimeType(TimeType.TimeDs50)
                    .setTask(task)
                    .build();

            // Execute propagation
            PropResponse response = stub.prop(request);

            long elapsedMs = System.currentTimeMillis() - startTime;

            // Print results
            EphemerisData result = response.getResult();
            System.out.println("=== api.v1.Propagator.Prop Response ===");
            System.out.println("Request ID:  " + response.getReqId());
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
