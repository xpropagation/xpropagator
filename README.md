## Menu

- [Overview](#overview)
- [Stack](#stack)
- [Why Not REST HTTP](#why-not-rest-http)
- [Examples](#examples)
- [API](#api)
    - [Prop](#prop)
    - [Ephem](#ephem)
    - [Info](#info)
- [Garbage Collection](#garbage-collection)  
- [Concurrency Model](#concurrency-model)
- [Configuration](#configuration)  
- [TLS](#tls)  
- [Testing](#testing)
- [Get USSF SGP4/SGP4-XP](#get-ussf-sgp4)
- [Docker](#docker)
  - [Linux AMD64](#linux-amd64)
  - [Linux ARM](#linux-arm)
- [Roadmap](#roadmap)
---

## Overview

**XPropagator**  
 Satellite orbit propagation gRPC service, offering an API for the U.S. Space Force (USSF) SGP4/SGP4-XP implementation. Written in Go/Cgo.    

- **Configurable server-side streaming**  
  Streamed gRPC responses for real-time propagation results or massive ephemeris datasets.

- **Ephemeris generation**  
  Single satellite propagation or large-scale ephemeris generation.

- **Mutual TLS security**  
  Optional end-to-end encryption with client + server certificate authentication.

- **Automatic resource management**  
  Built-in resource garbage collection eliminates manual tuning for production workloads.


| Aspect                 | Vallado SGP4 (Open-Source)                                                                                  | USSF SGP4-XP                                                                                                                                                                          |
|------------------------|-------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Accuracy (LEO/MEO)** | Good for short-term (days); errors ~km in GEO. Standardized fixes over legacy code improve LEO/MEO results. | Enhanced over classical SGP4; median errors under 0.66 km for 10 days in MEO/GEO.                                                                                                     |
| **Accuracy (GEO)**     | Meters-to-km errors; better than legacy but limited for long-term/high orbits.                              | Significantly improved (e.g., 500m vs. 5km errors); approaches Special Perturbations (SPEPH) fidelity.                                                                                |
| **Runtime**            | Faster baseline propagation.                                                                                | 50-100% longer than SGP4, but enables better velocity solutions and stale TLE usage.                                                                                                  |
| **Drag Model**         | Simplified ballistic coefficient (B*) approximation; basic density model (Jacchia/Robertson?).              | Explicit B* solving + AGOM drag term; more accurate atmospheric density handling for varying solar activity.                                                                          |
| **WGS/Gravity Model**  | WGS-72 (World Geodetic System 1972) Earth model; zonal harmonics J2-J6.                                     | Enhanced gravity modeling (likely WGS-84 compatible); improved RTC/REF epoch handling for TLE consistency.                                                                            |
| **RTC/REF Handling**   | Standard epoch conversion; limited deep-space drag compensation.                                            | Improved Reference Temperature Coefficient (RTC) and epoch adjustments for long-arc propagation stability.                                                                            |
| **Enhancements**       | Debugged public standard; WGS-72 gravity model.                                                             | Solves for B* and AGOM drag terms; tuned for MEO/GEO dynamics. Publicly released by USSF in 2020.                                                                                     |
| **Use Cases**          | General-purpose, short-arc predictions.                                                                     | Working with official Space-Track.org NORAD TLEs, MEO/GEO operational satellites, long-term predictions, production SSA pipelines requiring highest accuracy, mission-critical tasks. |

## Web Links for Sources

### USSF SGP4-XP References
| Citation | Title/Description                                                                                                                                                                                                | URL                                                                                                                   |
|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------|
| [web:1]  | Improved Orbital Predictions using Pseudo Observations - Maximizing the Utility of SGP4-XP (Anthony Holincheck, Janet Cathell, Sceptre Analytics, Inc, AMOS 2021)                                                | https://amostech.com/TechnicalPapers/2021/Astrodynamics/Holincheck.pdf [web:1]                                        |
| [web:2]  | Assessing Performance Characteristics of the SGP4-XP Propagation Algorithm (Dave Conkey, Mitchell Zielinski, a.i. solutions, AMOS 2022) - Primary technical analysis with accuracy/runtime data, drag/SRP models | https://amostech.com/TechnicalPapers/2022/Poster/Conkey.pdf [web:2]                                                   |
| [web:3]  | SGP4-XP Propagation Algorithm (a.i. solutions newsroom summary) - USSF release 2020, SPEPH-equivalent accuracy claims                                                                                            | https://ai-solutions.com/newsroom/assessing-performance-characteristics-of-the-sgp4-xp-propagation-algorithm/ [web:3] |

### Vallado SGP4 References
| Citation | Title/Description                                                                                                          | URL                                                                                        |
|----------|----------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------|
| [web:4]  | Comparison and Design of Simplified General Perturbation Models (SGP4) (Cal Poly thesis) - Vallado implementation analysis | https://digitalcommons.calpoly.edu/cgi/viewcontent.cgi?article=1094&context=theses [web:4] |
| [web:5]  | Comparison and Design of SGP4 (Cal Poly digital commons landing page)                                                      | https://digitalcommons.calpoly.edu/theses/86/ [web:5]                                      |

### Additional Supporting Sources
| Citation | Title/Description                                                 | URL                                               |
|----------|-------------------------------------------------------------------|---------------------------------------------------|
| [web:6]  | SGP4-XP: A Technical Leap in Satellite Tracking (keeptrack.space) | https://keeptrack.space/deep-dive/sgp4-xp [web:6] |
| [web:7]  | Space-Track.org Documentation                                     | https://www.space-track.org/documentation [web:7] |


> [!WARNING]
> This software is an independent open‑source project. It is **not affiliated with, or sponsored by** the U.S. Space Force (USSF).
> This repository **does not include or distribute** the official USSF SGP4/SGP4-XP binaries from space-track.org. Due to licensing restrictions, you must obtain them yourself directly from that source.
> Because of this, XPropagator cannot be distributed by us as a public Docker image or precompiled binary; you must build and deploy it manually for yourself.  
> See [Get USSF SGP4/SGP4-XP](#get-ussf-sgp4)

## Stack
- **Logging Pkg**: Uber's Zap  - https://github.com/uber-go/zap
- **ISO-8601 Time Duration Parser Pkg**: - https://github.com/sosodev/duration
- **USSF SGP4/SGP4-XP implementation** - https://www.space-track.org
- **Dependency Injection Framework**: Uber's Fx -https://github.com/uber-go/fx  
- **Remote Procedure Call Framework**: gRPC - https://github.com/grpc/grpc-go
- **Certificate Authority Server**: Smallstep CA - https://github.com/smallstep/certificates

## Why Not REST HTTP

| Aspect              | **gRPC, HTTP/2 (Current)**         | **REST, HTTP/1.1**               |
|---------------------|------------------------------------|----------------------------------|
| **Target Audience** | **SSA engineers, microservices**   | Web developers                   |
| **Ephem Streaming** | **Native**                         | **JSON chunks = nightmare**       |
| **Payload Size**    | Protobuf = **10x smaller**         | JSON = **gigabytes ephemeris**    |
| **Performance**     | **HTTP/2 + multiplexing**          | HTTP/1.1 = **queuing**            |
| **Security**        | **mTLS end-to-end**                | **TLS + JWT hack**                |
| **Integration**     | **Go/Python/C++/MATLAB**           | **curl/JS/Postman**               |


## Examples

> [!TIP]
> For more detailed examples, different use cases, and examples in other languages, see these: [Golang](/examples/code/go), [Python](/examples/code/py), [Java](/examples/code/java) [grpcurl](/examples/grpculr_requests_collection).
> If you want to contribute and add examples in new language, please see [CODE_EXAMPLES_CONTRIBUTING.md](CODE_EXAMPLES_CONTRIBUTING.md)

> [!TIP]
> It supports mutual TLS security out of the box, please see [examples](/examples) and [TLS](#tls)

> [!IMPORTANT]
> **TLS examples must be run from the project root directory** to find certificates in `scripts/certs/`.
> in example:
> ```bash
> cd {PROJECT ROOT}
> go run ./examples/code/go/generate_ephemeris/known_time_step_utc/secure
> python examples/code/py/single_propagate__utc_time_type.py
> ```

## API

> [!TIP]
> If you don't have gRPCurl installed, please follow https://github.com/fullstorydev/grpcurl?tab=readme-ov-file#installation
> To be able to invoke API via grpcurl please enable server reflection in the service configuration:
> ```yaml
> # config.yaml
> # the rest of config...
> 
> reflection: true
> 
> # the rest of config...
> ```

### Prop
Propagates the orbit of a specific satellite using its TLE to a given time. The time can be specified in UTC, as DS50 (days since January 1, 1950), or as MSE (minutes since the epoch)

**gRPCurl**

```bash
grpcurl -plaintext -d '{
    "req_id": "1",
    "task": {
      "time_utc": "2025-12-18T00:00:00Z",
      "sat": {
        "norad_id": 65271,
        "name": "X-37B Orbital Test Vehicle 8 (OTV 8)",
        "tle_ln1": "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
        "tle_ln2": "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05"
      }
    }
}' localhost:50051 api.v1.Propagator.Prop
```

Response
```json
{
  "reqId": "1",
  "result": {
    "ds50Time": 27744.5,
    "mseTime": 98117.24955839803,
    "x": 6632.395705215065,
    "y": -963.4385419237258,
    "z": -301.93019901550207,
    "vx": 0.9816963628539945,
    "vy": 4.993942889041923,
    "vz": 5.793776724520656
  }
}

```

**Go**

```go
package main

import (
  "context"
  apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"
  "google.golang.org/protobuf/types/known/timestamppb"
  "log"
  "time"
)

func main() {
  cc, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
  if err != nil {
    log.Fatal("Failed to create XPropagator client: ", err)
  }
  defer func() {
    _ = cc.Close()
  }()

  c := apiv1.NewPropagatorClient(cc)

  timeStart := time.Now()

  propReq := &apiv1.PropRequest{
    ReqId: 1,
    Task: &apiv1.PropTask{
      TimeUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
      Sat: &apiv1.Satellite{
        NoradId:  65271,
        Name:   "X-37B Orbital Test Vehicle 8 (OTV 8)",
        TleLn1: "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
        TleLn2: "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05",
      },
    },
  }

  resp, err := c.Prop(context.Background(), propReq)
  if err != nil {
    log.Fatalf("failed to request api.v1.Propagator.Prop: %s", err)
  }

  log.Printf("api.v1.Propagator.Prop done, time took: %s\n%s", time.Since(timeStart), resp.GetResult())
}

```

### Ephem
Generates ECI or J2K ephemeris data for multiple satellites using their TLEs over a specified time grid. The start and end times can be given in UTC or DS50 (days since January 1, 1950). The time step can be dynamic, and the time grid can be shared by all satellites or customized individually for each one.

This example generates ephemeris data for three satellite for the period from December 18, 2025, 00:00:00 UTC to
December 28, 2025, 00:00:00 UTC, with data points at 8.5-minute intervals.

'PT8.5M' corresponds 8.5 minutes interval in ISO-8601 duration format.
https://docs.digi.com/resources/documentation/digidocs/90001488-13/reference/r_iso_8601_duration_format.htm

**gRPCurl**

```bash
grpcurl -plaintext -d '{
    "req_id": "1",
    "ephem_type": "EphemJ2K",
    "common_time_grid": {
      "time_start_utc": "2025-12-18T00:00:00Z",
      "time_end_utc": "2025-12-28T00:00:00Z",
      "known_time_step_period": "PT8.5M"
    },
    "tasks": [
    {
      "task_id": 10,
      "sat": {
        "norad_id": 65271,
        "name": "X-37B Orbital Test Vehicle 8 (OTV 8)",
        "tle_ln1": "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
        "tle_ln2": "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05"
      }
    },
    {
      "task_id": 20,
      "sat": {
        "norad_id": 25544,
        "name": "International Space Station (ISS)",
        "tle_ln1": "1 25544U 98067A   26006.24223504  .00010833  00000-0  20347-3 0  9990",
        "tle_ln2": "2 25544  51.6328  21.1584 0007565 346.4032  13.6751 15.49129493546659"
      }
    },
    {
      "task_id": 30,
      "sat": {
        "norad_id": 23605,
        "name": "HELIOS 1A",
        "tle_ln1": "1 23605U 95033A   26006.14103603  .00004682  00000-0  33679-3 0  9999",
        "tle_ln2": "2 23605  98.2174 246.2520 0011561 210.6587 149.3963 15.04729092643785"
      }
    }
  ]
}' localhost:50051 api.v1.Propagator.Ephem
```

Response

// ... the rest of ephemeris data
```json
{
  "reqId": "1",
  "streamId": "2",
  "streamChunkId": "833",
  "result": {
    "taskId": "30",
    "ephemData": [
      {
        "ds50Time": 27754.3340277765,
        "x": 3973.1358986225705,
        "y": 5662.748137283377,
        "z": 426.13749504651963,
        "vx": 1.1416373159215192,
        "vy": -0.23650495922160086,
        "vz": -7.49710383456928
      },
      {
        "ds50Time": 27754.339930554277,
        "x": 3922.1154915336524,
        "y": 4688.189867280561,
        "z": -3266.1809243392167,
        "vx": -1.335920734072992,
        "vy": -3.484740202789356,
        "vz": -6.603656666918057
      },
      {
        "ds50Time": 27754.345833332052,
        "x": 2681.152784520029,
        "y": 2291.131068704907,
        "z": -5965.44143186784,
        "vx": -3.4026637206909074,
        "vy": -5.669397048813632,
        "vz": -3.7068734388111966
      }
    ],
    "ephemPointsCount": "3"
  }
}
```
// ...the rest of ephemeris data

**Go**

```go
package main

import (
  "context"
  apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"
  "google.golang.org/protobuf/types/known/timestamppb"
  "io"
  "log"
  "time"
)

func main() {
  cc, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
  if err != nil {
    log.Fatal("Failed to create XPropagator client: ", err)
  }
  defer func() {
    _ = cc.Close()
  }()

  c := apiv1.NewPropagatorClient(cc)

  timeStart := time.Now()

  ephemReq := &apiv1.EphemRequest{
    ReqId:     1,
    EphemType: apiv1.EphemType_EphemJ2K,
    CommonTimeGrid: &apiv1.EphemTimeGrid{
      TimeStartUtc: timestamppb.New(time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC)),
      TimeEndUtc:   timestamppb.New(time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)),
      TimeStepType: &apiv1.EphemTimeGrid_KnownTimeStepPeriod{
        KnownTimeStepPeriod: "PT8.5M",
      },
    },
    Tasks: []*apiv1.EphemTask{
      {
        TaskId: 10,
        Sat: &apiv1.Satellite{
          NoradId:  65271,
          Name:   "X-37B Orbital Test Vehicle 8 (OTV 8)",
          TleLn1: "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
          TleLn2: "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05",
        },
      },
      {
        TaskId: 20,
        Sat: &apiv1.Satellite{
          NoradId:  25544,
          Name:   "International Space Station (ISS)",
          TleLn1: "1 25544U 98067A   26006.24223504  .00010833  00000-0  20347-3 0  9990",
          TleLn2: "2 25544  51.6328  21.1584 0007565 346.4032  13.6751 15.49129493546659",
        },
      },
      {
        TaskId: 30,
        Sat: &apiv1.Satellite{
          NoradId:  23605,
          Name:   "HELIOS 1A",
          TleLn1: "1 23605U 95033A   26006.14103603  .00004682  00000-0  33679-3 0  9999",
          TleLn2: "2 23605  98.2174 246.2520 0011561 210.6587 149.3963 15.04729092643785",
        },
      },
    },
  }

  ctx := context.Background()

  stream, err := c.Ephem(ctx, ephemReq)
  if err != nil {
    log.Fatalf("failed to request api.v1.Propagator.Ephem: %s", err)
  }
  for {
    resp, err := stream.Recv()
    if err == io.EOF {
      break
    }
    if err != nil {
      log.Fatalf("failed to receive stream chunk from api.v1.Propagator.Ephem: %s", err)
    }

    log.Printf("api.v1.Propagator.Ephem stream chunk received:\n"+
            "ReqId: %d\n"+
            "TaskId: %d\n"+
            "StreamId: %d\n"+
            "StreamChunkId: %d\n"+
            "EphemerisCount: %v",
      resp.GetReqId(),
      resp.GetResult().GetTaskId(),
      resp.GetStreamId(),
      resp.GetStreamChunkId(),
      resp.GetResult().EphemPointsCount,
    )

    for _, ephemData := range resp.GetResult().EphemData {
      log.Println(ephemData.String())
    }
  }

  log.Printf("api.v1.Propagator.Ephem done, time took: %s", time.Since(timeStart))
}
```

### Info
Provides general information about service.

**gRPCurl**

```bash
grpcurl -plaintext localhost:50051 api.v1.Propagator.Info
```

Response
```text
{
  "name": "XPropagator Server",
  "version": "v1.0.0",
  "commit": "f0d0d0d2bc943b76d58c49ae0dfac615cd195d28",
  "buildDate": "2025-10-17T12:45:21Z",
  "astroStdLibInfo": "HQ SpOC DllMain - Version: v9.6 - Build: Jun 11 2025 - Platform: M1 Mac 64-bit - Compiler: gfortran",
  "sgp4LibInfo": "HQ SpOC Sgp4Prop - Version: v9.6 - Build: Jun 11 2025 - Platform: M1 Mac 64-bit - Compiler: gfortran",
  "timestamp": "2025-10-17T12:45:27.147136Z"
}
```

**Go**

```go
package main

import (
  "context"
  apiv1 "github.com/xpropagation/xpropagator/api/v1/gen"
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"
  "google.golang.org/protobuf/types/known/emptypb"
  "log"
)

func main() {
  cc, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
  if err != nil {
    log.Fatal("Failed to create XPropagator client: ", err)
  }
  defer func() {
    _ = cc.Close()
  }()

  c := apiv1.NewPropagatorClient(cc)

  infoResp, err := c.Info(context.Background(), &emptypb.Empty{})
  if err != nil {
    log.Fatal("failed to request api.v1.Propagator.Info : ", err)
  }

  log.Printf("api.v1.Propagator.Info response:\nName: %s,\nVersion: %s,\nCommit: %s,\nBuildDate: %s,\nAstroStdLibInfo: %s,\nSgp4LibInfo: %s,\nTimestamp: %s",
    infoResp.GetName(), infoResp.GetVersion(), infoResp.GetCommit(), infoResp.GetBuildDate(), infoResp.GetAstroStdLibInfo(), infoResp.GetSgp4LibInfo(), infoResp.GetTimestamp().AsTime().String())
}
```

## Garbage Collection

**Custom LRU+TTL cache GC** for Sat/TLE objects. Controls memory usage during high-volume TLE ingestion.

### Key Features
- **Size limit**: `max_loaded_sats_gc` — max concurrent satellites
- **Idle eviction**: `idle_ttl_gc_min` — TTL for unused propagators
- **Sweep frequency**: `sweep_interval_gc_min` — GC cycle interval

### Acquire/Release Pattern
1) Acquire(TLE) → satKey + release func (refcount++)

2) Use SGP4 propagation with satKey

3) Call release() (refcount--)

### Eviction Strategy
- **LRU**: Evicts least-recently-used (refs==0) when size exceeded
- **TTL**: Removes idle satellites (refs==0 + past TTL) during sweeps
- **Safe removal**: Double-checked locks prevent in-use deletion

### Thread Safety
Per-satellite RWMutex + global catalog lock + GC mutex.

Zero-copy reference counting ensures propagators stay alive during active propagation requests.

## Concurrency Model

XPropagator processes requests **sequentially by design** using a global mutex lock.

### Why Sequential?

The underlying USSF SGP4 Fortran DLL uses global state and is **not thread-safe**. Concurrent calls would corrupt data or crash. The global lock ensures safe, correct operation.

### Request Queuing Behavior

| Scenario | Behavior |
|----------|----------|
| Multiple simultaneous requests | All block, processed one at a time |
| Request during Ephem stream | Waits until stream completes |
| Processing order | Not guaranteed (not FIFO) |
| Wait timeout | Indefinite until lock available |

```
Timeline (3 clients arrive during active request):

Client 1: [════ PROCESSING ════]
Client 2:          ░░ WAIT ░░░░░[═══ PROCESS ═══]
Client 3:          ░░ WAIT ░░░░░░░░░░░░░░░░░░░░░[═══ PROCESS ═══]
Client 4:          ░░ WAIT ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░[═══]
──────────────────────────────────────────────────────────────────────►
```

### Performance Implications

- **Throughput**: Limited to one DLL call at a time
- **Latency**: Wait time = sum of all preceding request durations
- **Scaling**: For high-throughput scenarios, deploy multiple XPropagator instances behind a load balancer.

## TLS
To enable end-to-end strong TLS encryption with mutual certificate authentication (mTLS) between client and server, configure the TLS settings in your service configuration as shown below:
```yaml
# config.yaml
# the rest of config...

tls:
  enabled: true
  cert_file: "your/path/to/the/certs/server.crt"
  key_file: "your/path/to/the/certs/server.key"
  ca_file: "your/path/to/the/certs/ca.crt"
  
# the rest of config...
```

Or set these ENV variables:
`SERVICE_ENABLE_TLS`, `SERVICE_TLS_CERT_FILE_PATH`, `SERVICE_TLS_KEY_FILE_PATH`, `SERVICE_TLS_CA_FILE_PATH`

> [!NOTE]
> For quick local development we use Smallstep CA as our Certificate Authority (CA). To read more see https://smallstep.com/docs/step-ca/

> [!TIP]
> if you want to invoke gRPC API with TLS using grpcurl CLI tool in your local environment, please do this before:
>
> ```bash
> echo "127.0.0.1 xpropagator-server" | sudo tee -a /etc/hosts
> ```

## Configuration
Service configuration supports both environment variables (ENV) and YAML files, with ENV taking highest priority.

Configuration Priority Order
 - ENV variables (highest priority) - completely override matching YAML values
 - YAML file (fallback when ENV vars are absent)

Default configuration YAML file may look like this:

```yaml
reflection: true
stream_chunk_size: 3
graceful_stop_timeout_sec: 10s
gc:
  max_loaded_sats_gc: 10000
  idle_ttl_gc_min: 20m
  sweep_interval_gc_min: 2m
tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"
  ca_file: "certs/ca.crt"

```

| YAML Property             | ENV Variable                      | Description                                                                                                                                                                                                                            | Default Value           |
|---------------------------|-----------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------|
| none                      | SERVICE_CONFIG                    | Path to the configuration YAML file                                                                                                                                                                                                    | config/cfg_default.yaml |
| none                      | SERVICE_HOST                      | gRPC Server host                                                                                                                                                                                                                       | 0.0.0.0                 |
| none                      | SERVICE_PORT                      | gRPC Server port                                                                                                                                                                                                                       | 50051                   |
| reflection                | SERVICE_REFLECTION                | Enable gRPC server reflection allows clients to dynamically discover a server's available services, methods, and protobuf message schemas at runtime without precompiled stubs, primarily for debugging tools like grpcurl and Postman | false                   |
| stream_chunk_size         | SERVICE_STREAM_CHUNK_SIZE         | Control the number of chunks in ephemeris data and the number of 6D state vectors per each chunk                                                                                                                                       | 100                     |
| graceful_stop_timeout_sec | SERVICE_GRACEFUL_STOP_TIMEOUT_SEC | Sets the maximum time (in seconds) the service waits for ephemeris processing to complete cleanly before forcing shutdown                                                                                                              | 10                      |
| gc.max_loaded_sats_gc     | SERVICE_MAX_LOADED_SATS_GC        | Maximum satellites loaded simultaneously before GC eviction kicks in. Prevents memory bloat from TLE/Sat caches                                                                                                                        | 500                     |
| gc.idle_ttl_gc_min        | SERVICE_IDLE_TTL_GC_MIN           | Time-to-live for idle satellites before GC removal (minutes). Frees unused TLE/Sat                                                                                                                                                     | 10                      |
| gc.sweep_interval_gc_min  | SERVICE_SWEEP_INTERVAL_GC_MIN     | GC sweep frequency (minutes). Scans memory pool for idle/expired satellite data                                                                                                                                                        | 5                       |
| tls.enabled               | SERVICE_ENABLE_TLS                | Enable end-to-end encrypted connection with client + server certificate authentication using mutual TLS                                                                                                                                | false                   |
| tls.cert_file             | SERVICE_TLS_CERT_FILE_PATH        | Path to the server cert .crt file                                                                                                                                                                                                      | certs/server.crt        |
| tls.key_file              | SERVICE_TLS_KEY_FILE_PATH         | Path to the server key .key file                                                                                                                                                                                                       | certs/server.key        |
| tls.ca_file               | SERVICE_TLS_CA_FILE_PATH          | Path to the root CA .crt file                                                                                                                                                                                                          | certs/ca.crt            |

## Testing

XPropagator includes a test suite covering configuration, garbage collection, and API functionality.

> [!IMPORTANT]
> Unit and integration tests that interact with the SGP4 DLL require the libraries to be installed in `/usr/local/lib/`. Run the build script first to install the libraries.

### Test Files

| Test File | Package | Description |
|-----------|---------|-------------|
| `internal/config/config_test.go` | config | Unit tests for configuration loading |
| `internal/core/gc/gc_test.go` | gc | Unit tests for garbage collection |
| `internal/core/api_integration_test.go` | core | Integration tests for gRPC API |

### Wrapper Scripts

| Script | Description | DLL Required |
|--------|-------------|--------------|
| `scripts/run_unit_tests_config.sh` | Runs config package unit tests |  No |
| `scripts/run_unit_tests_gc.sh` | Runs GC package unit tests |  Yes |
| `scripts/run_integration_tests_core.sh` | Runs core API integration tests |  Yes |

### Test Coverage

#### Config Tests (`internal/config/config_test.go`)
Tests configuration loading from YAML files and environment variables:
- Default values when no config file exists
- YAML file parsing and loading
- Environment variable overrides (ENV takes priority over YAML)
- Invalid/malformed YAML handling
- Zero/negative value handling
- TLS enable/disable logic
- Duration parsing (minutes, hours, seconds, milliseconds)
- `Config.String()` output formatting

#### GC Tests (`internal/core/gc/gc_test.go`)
Tests the custom LRU+TTL garbage collection for satellite objects:
- Default and custom GC configuration values
- Max satellites limit and LRU eviction ordering
- Protection of in-use satellites (refs > 0) from eviction
- TTL-based idle entry identification
- Releaser function (reference counting, lastUsed updates)
- Sweeper goroutine lifecycle
- `WaitAllReleased` with timeout handling
- Concurrent access safety
- TLE satellite number parsing
- Benchmarks for eviction and releaser performance

#### API Integration Tests (`internal/core/api_integration_test.go`)
Tests the gRPC Propagator API endpoints with real DLL calls:
- **Info API**: Service information retrieval
- **Prop API**: Single satellite propagation
  - DS50, UTC, and MSE time types
  - Multiple satellites in sequence
  - Invalid TLE error handling
- **Ephem API**: Ephemeris generation (streaming)
  - ECI and J2K reference frames
  - Single and multiple satellites
  - Common and individual time grids
  - Dynamic time steps
  - Various ISO-8601 duration formats
  - Long time ranges (full day)
  - High-frequency sampling
  - Stream metadata verification
  - Context cancellation handling
- **Concurrency Tests**: Global lock behavior verification
  - Concurrent Prop requests
  - Concurrent Ephem requests (same/different satellites)
  - Concurrent Ephem with multiple tasks per request
  - Stress test with 10 concurrent requests
  - Mixed Prop and Ephem concurrent requests

### Running Tests

```bash
cd scripts

# Run config tests (no DLL required)
bash ./run_unit_tests_config.sh

# Run GC tests (requires libraries in /usr/local/lib/)
bash ./run_unit_tests_gc.sh

# Run integration tests (requires libraries in /usr/local/lib/)
bash ./run_integration_tests_core.sh
```

## Get USSF SGP4
The USSF version of SGP4/SGP4-XP available through Space-Track.org is distributed as compiled binaries rather than source code primarily to protect proprietary enhancements and operational details used in military satellite tracking. This approach ensures precise compatibility with TLEs generated by USSF systems while preventing reverse-engineering or unintended modifications that could lead to inconsistencies in orbital predictions. Earlier public releases, like Spacetrack Report #3, provided open FORTRAN code, but modern variants such as SGP4-XP incorporate classified perturbations for improved accuracy, justifying the closed-source format.

Registered users on Space-Track.org can download the AstroStds library binaries (version 8.x and later) with non-restricted access, but source code remains unavailable. This binary-only distribution started around 2020 alongside SGP4-XP to address discrepancies between public implementations and USSF operations.  

  1. Create an account at [Space‑Track](https://www.space-track.org/auth/createAccount).
  2. Sign in and navigate to the [SGP4 documentation](https://www.space-track.org/documentation#/sgp4).
  3. Download `Sgp4Prop_vX.Y.zip` archive.
  4. Extract `Sgp4Prop_vX.Y.zip` archive to eny destination you want, let's say {SGP4 ROOT}.

## Docker

This shell script executes unit and integration tests, installs local Smallstep TLS certificates (if enabled), compiles the binary for your target Linux architecture, builds a Docker image, and runs the container.

> [!IMPORTANT]
> TLS is disabled by default. To produce a build with enabled TLS encryption, pass 'true' as the final argument as `build_and_run_docker_linux.sh {SGP4_LIB_PATH} {SGP4_WRAPPERS_PATH} true`

### Linux AMD64
```bash
cd {REPO ROOT}/scripts
chmod +x build_and_run_docker_linux.sh
docker rm xpropagator-server
rm -rf ./certs
./build_and_run_docker_linux.sh /{SGP4 ROOT}/Lib/Linux/GFORTRAN /{SGP4 ROOT}/SampleCode/Go/DriverExamples/wrappers
```

### Linux ARM
```bash
cd {REPO ROOT}/scripts
chmod +x build_and_run_docker_linux.sh
docker rm xpropagator-server
rm -rf ./certs
./build_and_run_docker_linux.sh /{SGP4 ROOT}/Lib/Linux_ARM/GFORTRAN /{SGP4 ROOT}/SampleCode/Go/DriverExamples/wrappers
```

---

## Roadmap

Future improvements (contributions welcome!):

- [ ] **REST HTTP Gateway** — Implement REST/JSON API gateway (only if there is demand from users)
- [ ] **Benchmarks** - Write Vallado SGP4 vs USSF SGP4 vs USSF SGP4-XP benchmarks
- [ ] **More Language Examples** — Add code examples in Rust, C++, MATLAB, and other languages
- [ ] **Extended Test Coverage** — Write more unit tests for edge cases and error handling

---

## Contributing

Contributions are welcome! Whether it's bug fixes, new features, documentation improvements, or code examples in new languages — all PRs are appreciated.

### Contribution Ideas

- Add code examples in new languages (Rust, C++, MATLAB, etc.)
- Improve documentation
- Report bugs or suggest features via GitHub Issues
- Performance optimizations
- Additional test coverage

For adding code examples in new languages, please see [CODE_EXAMPLES_CONTRIBUTING.md](CODE_EXAMPLES_CONTRIBUTING.md).

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.