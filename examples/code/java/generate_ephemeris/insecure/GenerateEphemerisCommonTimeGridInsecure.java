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

package generate_ephemeris.insecure;

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
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.util.Iterator;

/**
 * Example demonstrating ephemeris generation for multiple satellites using a common time grid
 * with an insecure (plaintext) connection.
 *
 * Common time grid means all satellites share the same start/end times and time step.
 * This example uses dynamic time step (SGP4-optimized intervals).
 *
 * WARNING: This should only be used for local development/testing.
 * In production, always use mTLS (see GenerateEphemerisCommonTimeGridSecure.java).
 */
public class GenerateEphemerisCommonTimeGridInsecure {

    private static final String SERVER_HOST = "localhost";
    private static final int SERVER_PORT = 50051;

    public static void main(String[] args) {
        ManagedChannel channel = null;
        try {
            // Build insecure channel (plaintext, no TLS)
            channel = ManagedChannelBuilder.forAddress(SERVER_HOST, SERVER_PORT)
                    .usePlaintext()
                    .build();

            // Create blocking stub for synchronous streaming
            PropagatorGrpc.PropagatorBlockingStub stub = PropagatorGrpc.newBlockingStub(channel);

            long startTime = System.currentTimeMillis();

            // Build common time grid: 10 days with dynamic time step
            ZonedDateTime timeStart = ZonedDateTime.of(2025, 12, 18, 0, 0, 0, 0, ZoneOffset.UTC);
            ZonedDateTime timeEnd = ZonedDateTime.of(2025, 12, 28, 0, 0, 0, 0, ZoneOffset.UTC);

            EphemTimeGrid commonTimeGrid = EphemTimeGrid.newBuilder()
                    .setTimeStartUtc(toTimestamp(timeStart))
                    .setTimeEndUtc(toTimestamp(timeEnd))
                    .setDynamicTimeStep(true)  // SGP4-optimized intervals
                    .build();

            // Build satellite tasks
            EphemTask task1 = EphemTask.newBuilder()
                    .setTaskId(10)
                    .setSat(Satellite.newBuilder()
                            .setNoradId(65271)
                            .setName("X-37B Orbital Test Vehicle 8 (OTV 8)")
                            .setTleLn1("1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07")
                            .setTleLn2("2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05")
                            .build())
                    .build();

            EphemTask task2 = EphemTask.newBuilder()
                    .setTaskId(20)
                    .setSat(Satellite.newBuilder()
                            .setNoradId(2000)
                            .setName("Satellite B")
                            .setTleLn1("1 49220U 21089B   24290.21456789  .00014567  00000-0  62458-3 0  9991")
                            .setTleLn2("2 49220  53.0021 320.8765 0078456  42.6543 317.8845 14.87654321987654")
                            .build())
                    .build();

            EphemTask task3 = EphemTask.newBuilder()
                    .setTaskId(30)
                    .setSat(Satellite.newBuilder()
                            .setNoradId(3000)
                            .setName("Satellite C")
                            .setTleLn1("1 60123U 24150C   24290.84567890  .00000023  00000-0  15987-5 0  9993")
                            .setTleLn2("2 60123  28.5123 210.2345 0019876 102.3456 257.6543 12.34567890123456")
                            .build())
                    .build();

            // Build ephemeris request with common time grid
            EphemRequest request = EphemRequest.newBuilder()
                    .setReqId(1)
                    .setEphemType(EphemType.EphemJ2K)  // J2000 reference frame
                    .setCommonTimeGrid(commonTimeGrid)
                    .addTasks(task1)
                    .addTasks(task2)
                    .addTasks(task3)
                    .build();

            // Execute streaming ephemeris generation
            System.out.println("=== api.v1.Propagator.Ephem Streaming Response ===");
            System.out.println("Time grid: " + timeStart + " to " + timeEnd);
            System.out.println("Satellites: 3");
            System.out.println();

            Iterator<EphemResponse> responseIterator = stub.ephem(request);
            int totalPoints = 0;

            while (responseIterator.hasNext()) {
                EphemResponse response = responseIterator.next();
                EphemOut result = response.getResult();

                System.out.printf("Stream chunk received: ReqId=%d, TaskId=%d, StreamId=%d, ChunkId=%d, Points=%d%n",
                        response.getReqId(),
                        result.getTaskId(),
                        response.getStreamId(),
                        response.getStreamChunkId(),
                        result.getEphemPointsCount());

                totalPoints += result.getEphemPointsCount();

                // Print first and last ephemeris point of each chunk
                if (result.getEphemDataCount() > 0) {
                    EphemerisData first = result.getEphemData(0);
                    System.out.printf("  First point: DS50=%.6f, X=%.3f, Y=%.3f, Z=%.3f%n",
                            first.getDs50Time(), first.getX(), first.getY(), first.getZ());

                    if (result.getEphemDataCount() > 1) {
                        EphemerisData last = result.getEphemData(result.getEphemDataCount() - 1);
                        System.out.printf("  Last point:  DS50=%.6f, X=%.3f, Y=%.3f, Z=%.3f%n",
                                last.getDs50Time(), last.getX(), last.getY(), last.getZ());
                    }
                }
                System.out.println();
            }

            long elapsedMs = System.currentTimeMillis() - startTime;
            System.out.println("=== Summary ===");
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
