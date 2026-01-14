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
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

import java.util.Iterator;

/**
 * Example demonstrating ephemeris generation with a known (fixed) time step
 * specified in DS50 days.
 *
 * DS50 is "days since 1950 Jan 0.0 UTC" - a common astrodynamics time format.
 * Time step of 0.005902777... days ≈ 8.5 minutes.
 *
 * This is the DS50 equivalent of the UTC known time step example.
 *
 * WARNING: This should only be used for local development/testing.
 * In production, always use mTLS.
 */
public class GenerateEphemerisKnownTimeStepDs50Insecure {

    private static final String SERVER_HOST = "localhost";
    private static final int SERVER_PORT = 50051;

    // 8.5 minutes in days: 8.5 / (24 * 60) = 0.005902777...
    private static final double TIME_STEP_DS50 = 0.005902777777777778;

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

            // Build time grid with DS50 times and known time step
            double timeStartDs50 = 27744.5;  // Dec 18, 2025
            double timeEndDs50 = 27754.5;    // Dec 28, 2025

            EphemTimeGrid timeGrid = EphemTimeGrid.newBuilder()
                    .setTimeStartDs50(timeStartDs50)
                    .setTimeEndDs50(timeEndDs50)
                    .setKnownTimeStepDs50(TIME_STEP_DS50)  // 8.5 minutes in days
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
            System.out.println("=== api.v1.Propagator.Ephem Known Time Step (DS50) ===");
            System.out.printf("Time range: DS50 %.6f to %.6f (10 days)%n", timeStartDs50, timeEndDs50);
            System.out.printf("Time step:  %.15f days (≈ 8.5 minutes)%n", TIME_STEP_DS50);
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
}
