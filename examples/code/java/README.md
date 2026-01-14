# Java Examples

Java examples for interacting with the XPropagator gRPC API.

## Prerequisites

- Java 11 or later
- Gradle 7.x+ (or use the Gradle wrapper)

## Project Structure

## Quick Start

### Using Gradle

Gradle automatically generates Java classes from proto files:

```bash
# Generate proto classes and build
./gradlew build

./gradlew runInfoSecure
./gradlew runInfoInsecure

./gradlew runSinglePropagateDs50TimeTypeSecure
./gradlew runSinglePropagateDs50TimeTypeInsecure

./gradlew runSinglePropagateMseTimeTypeSecure
./gradlew runSinglePropagateMseTimeTypeInsecure

./gradlew runSinglePropagateUtcTimeTypeSecure
./gradlew runSinglePropagateUtcTimeTypeInsecure

./gradlew runGenerateEphemerisCommonTimeGridSecure
./gradlew runGenerateEphemerisCommonTimeGridInsecure

./gradlew runGenerateEphemerisMixedTimeGridSecure
./gradlew runGenerateEphemerisMixedTimeGridInsecure

./gradlew runGenerateEphemerisKnownTimeStepUtcSecure
./gradlew runGenerateEphemerisKnownTimeStepUtcInsecure

./gradlew runGenerateEphemerisKnownTimeStepDs50Secure
./gradlew runGenerateEphemerisKnownTimeStepDs50Insecure

./gradlew runGenerateEphemerisJ2kFrameSecure
./gradlew runGenerateEphemerisJ2kFrameInsecure

./gradlew runGenerateEphemerisEciFrameSecure
./gradlew runGenerateEphemerisEciFrameInsecure

./gradlew runGenerateEphemerisDs50TimeTypeSecure
./gradlew runGenerateEphemerisDs50TimeTypeInsecure

./gradlew runGenerateEphemerisUtcTimeTypeSecure
./gradlew runGenerateEphemerisUtcTimeTypeInsecure
```