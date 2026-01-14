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

package generate_ephemeris.secure;

import api.v1.Common.EphemerisData;
import api.v1.Common.Satellite;
import api.v1.Ephem.EphemOut;
import api.v1.Ephem.EphemRequest;
import api.v1.Ephem.EphemResponse;
import api.v1.Ephem.EphemTask;
import api.v1.Ephem.EphemTimeGrid;
import api.v1.Ephem.EphemType;
import api.v1.PropagatorGrpc;
import com.google.protobuf.Timestamp;
import helper.TlsHelper;
import io.grpc.ManagedChannel;
import io.grpc.netty.shaded.io.grpc.netty.NettyChannelBuilder;
import io.grpc.netty.shaded.io.netty.handler.ssl.SslContext;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.util.Iterator;

/**
 * Example demonstrating ephemeris generation with a known (fixed) time step
 * specified as an ISO-8601 duration string, using mTLS.
 *
 * This generates ephemeris at regular intervals (e.g., every 8.5 minutes)
 * instead of dynamic SGP4-optimized intervals.
 *
 * ISO-8601 Duration Format Examples:
 * - "PT8.5M"  = 8.5 minutes
 * - "PT1H"    = 1 hour
 * - "PT30S"   = 30 seconds
 * - "P1D"     = 1 day
 *
 * See: https://en.wikipedia.org/wiki/ISO_8601#Durations
 */
public class GenerateEphemerisKnownTimeStepUtcSecure {

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

            // Create blocking stub for synchronous streaming
            PropagatorGrpc.PropagatorBlockingStub stub = PropagatorGrpc.newBlockingStub(channel);

            long startTime = System.currentTimeMillis();

            // Build time grid with known time step: 8.5 minutes interval
            ZonedDateTime timeStart = ZonedDateTime.of(2025, 12, 18, 0, 0, 0, 0, ZoneOffset.UTC);
            ZonedDateTime timeEnd = ZonedDateTime.of(2025, 12, 28, 0, 0, 0, 0, ZoneOffset.UTC);
            String timeStepPeriod = "PT8.5M";  // ISO-8601: 8.5 minutes

            EphemTimeGrid timeGrid = EphemTimeGrid.newBuilder()
                    .setTimeStartUtc(toTimestamp(timeStart))
                    .setTimeEndUtc(toTimestamp(timeEnd))
                    .setKnownTimeStepPeriod(timeStepPeriod)  // Fixed interval
                    .build();

            // Build satellite task
            EphemTask task = EphemTask.newBuilder()
                    .setTaskId(10)
                    .setSat(Satellite.newBuilder()
                            .setNoradId(65271)
                            .setName("X-37B Orbital Test Vehicle 8 (OTV 8)")
                            .setTleLn1("1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07")
                            .setTleLn2("2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05")
                            .build())
                    .build();

            // Build ephemeris request
            EphemRequest request = EphemRequest.newBuilder()
                    .setReqId(1)
                    .setEphemType(EphemType.EphemJ2K)
                    .setCommonTimeGrid(timeGrid)
                    .addTasks(task)
                    .build();

            // Execute streaming ephemeris generation
            System.out.println("=== api.v1.Propagator.Ephem Known Time Step (UTC) ===");
            System.out.println("Time range: " + timeStart + " to " + timeEnd);
            System.out.println("Time step:  " + timeStepPeriod + " (8.5 minutes)");
            System.out.println();

            Iterator<EphemResponse> responseIterator = stub.ephem(request);
            int totalPoints = 0;
            int chunkCount = 0;

            while (responseIterator.hasNext()) {
                EphemResponse response = responseIterator.next();
                EphemOut result = response.getResult();

                chunkCount++;
                totalPoints += result.getEphemPointsCount();

                System.out.printf("Stream chunk %d: TaskId=%d, Points=%d%n",
                        response.getStreamChunkId(),
                        result.getTaskId(),
                        result.getEphemPointsCount());

                // Print all ephemeris points in this chunk
                for (EphemerisData ephemData : result.getEphemDataList()) {
                    System.out.printf("  DS50=%.6f, X=%.3f, Y=%.3f, Z=%.3f, VX=%.6f, VY=%.6f, VZ=%.6f%n",
                            ephemData.getDs50Time(),
                            ephemData.getX(), ephemData.getY(), ephemData.getZ(),
                            ephemData.getVx(), ephemData.getVy(), ephemData.getVz());
                }
            }

            long elapsedMs = System.currentTimeMillis() - startTime;
            System.out.println();
            System.out.println("=== Summary ===");
            System.out.println("Total chunks: " + chunkCount);
            System.out.println("Total ephemeris points: " + totalPoints);
            System.out.println("Time elapsed: " + elapsedMs + " ms");

        } catch (Exception e) {
            System.err.println("Failed to call api.v1.Propagator.Ephem: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        } finally {
            if (channel != null) {
                channel.shutdownNow();
            }
        }
    }

    private static Timestamp toTimestamp(ZonedDateTime zdt) {
        Instant instant = zdt.toInstant();
        return Timestamp.newBuilder()
                .setSeconds(instant.getEpochSecond())
                .setNanos(instant.getNano())
                .build();
    }
}
