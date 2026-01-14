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
 * Example demonstrating ephemeris generation with mixed time grids:
 * - Some satellites use the common time grid
 * - Other satellites have individual time grids
 *
 * This is useful when different satellites need ephemeris for different time periods.
 *
 * WARNING: This should only be used for local development/testing.
 * In production, always use mTLS (see GenerateEphemerisMixedTimeGridSecure.java).
 */
public class GenerateEphemerisMixedTimeGridInsecure {

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

            // Build common time grid: Dec 18-28 with dynamic time step
            ZonedDateTime commonStart = ZonedDateTime.of(2025, 12, 18, 0, 0, 0, 0, ZoneOffset.UTC);
            ZonedDateTime commonEnd = ZonedDateTime.of(2025, 12, 28, 0, 0, 0, 0, ZoneOffset.UTC);

            EphemTimeGrid commonTimeGrid = EphemTimeGrid.newBuilder()
                    .setTimeStartUtc(toTimestamp(commonStart))
                    .setTimeEndUtc(toTimestamp(commonEnd))
                    .setDynamicTimeStep(true)
                    .build();

            // Task 1: Uses common time grid (no individual time_grid set)
            EphemTask task1 = EphemTask.newBuilder()
                    .setTaskId(10)
                    .setSat(Satellite.newBuilder()
                            .setNoradId(65271)
                            .setName("X-37B Orbital Test Vehicle 8 (OTV 8)")
                            .setTleLn1("1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07")
                            .setTleLn2("2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05")
                            .build())
                    .build();

            // Task 2: Uses INDIVIDUAL time grid (Dec 29-30)
            ZonedDateTime task2Start = ZonedDateTime.of(2025, 12, 29, 0, 0, 0, 0, ZoneOffset.UTC);
            ZonedDateTime task2End = ZonedDateTime.of(2025, 12, 30, 0, 0, 0, 0, ZoneOffset.UTC);

            EphemTask task2 = EphemTask.newBuilder()
                    .setTaskId(20)
                    .setTimeGrid(EphemTimeGrid.newBuilder()
                            .setTimeStartUtc(toTimestamp(task2Start))
                            .setTimeEndUtc(toTimestamp(task2End))
                            .setDynamicTimeStep(true)
                            .build())
                    .setSat(Satellite.newBuilder()
                            .setNoradId(2000)
                            .setName("Satellite B")
                            .setTleLn1("1 49220U 21089B   24290.21456789  .00014567  00000-0  62458-3 0  9991")
                            .setTleLn2("2 49220  53.0021 320.8765 0078456  42.6543 317.8845 14.87654321987654")
                            .build())
                    .build();

            // Task 3: Uses common time grid (no individual time_grid set)
            EphemTask task3 = EphemTask.newBuilder()
                    .setTaskId(30)
                    .setSat(Satellite.newBuilder()
                            .setNoradId(3000)
                            .setName("Satellite C")
                            .setTleLn1("1 60123U 24150C   24290.84567890  .00000023  00000-0  15987-5 0  9993")
                            .setTleLn2("2 60123  28.5123 210.2345 0019876 102.3456 257.6543 12.34567890123456")
                            .build())
                    .build();

            // Build ephemeris request with mixed time grids
            EphemRequest request = EphemRequest.newBuilder()
                    .setReqId(1)
                    .setEphemType(EphemType.EphemJ2K)
                    .setCommonTimeGrid(commonTimeGrid)
                    .addTasks(task1)
                    .addTasks(task2)
                    .addTasks(task3)
                    .build();

            // Execute streaming ephemeris generation
            System.out.println("=== api.v1.Propagator.Ephem Mixed Time Grid ===");
            System.out.println("Common time grid: " + commonStart + " to " + commonEnd);
            System.out.println("Task 10 (X-37B): uses common time grid");
            System.out.println("Task 20 (Sat B): uses individual time grid " + task2Start + " to " + task2End);
            System.out.println("Task 30 (Sat C): uses common time grid");
            System.out.println();

            Iterator<EphemResponse> responseIterator = stub.ephem(request);
            int totalPoints = 0;

            while (responseIterator.hasNext()) {
                EphemResponse response = responseIterator.next();
                EphemOut result = response.getResult();

                System.out.printf("Stream chunk: TaskId=%d, StreamId=%d, ChunkId=%d, Points=%d%n",
                        result.getTaskId(),
                        response.getStreamId(),
                        response.getStreamChunkId(),
                        result.getEphemPointsCount());

                totalPoints += result.getEphemPointsCount();

                // Print first ephemeris point of each chunk
                if (result.getEphemDataCount() > 0) {
                    EphemerisData first = result.getEphemData(0);
                    System.out.printf("  First: DS50=%.6f, X=%.3f, Y=%.3f, Z=%.3f%n",
                            first.getDs50Time(), first.getX(), first.getY(), first.getZ());
                }
            }

            long elapsedMs = System.currentTimeMillis() - startTime;
            System.out.println();
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
