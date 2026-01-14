We welcome contributions to this project, including new examples in additional programming languages. When adding examples in a new language, ensure comprehensive coverage of all required use cases.

## Contribution Guidelines
Please follow these steps for clear, high-quality submissions:

1. **Fork the repository** and create your feature branch from `main`.
2. **Add examples** in the new language under `/examples/[language]/`.
3. **Cover all use cases** listed below—no partial implementations.
4. **Add complete script(s)** for .proto gRPC API generation.  
5. **Update README** with new language badge and link.
6. **Submit pull request** with clear description of changes.

## Required Use Cases
All examples must demonstrate these scenarios:

> [!NOTE]
> You have complete flexibility over folder structure and example source file names within your language folder, but please keep the names of the use cases.

> [!WARNING]  
> **TLS Coverage Required**  
> All examples **must** demonstrate **both** connection types:
> - ✅ **Secure** (mTLS)
> - ✅ **Insecure** (TLS-free)

### Generate Ephemeris (api.v1.Propagator.Prop)
- ✅ Common Time Grid 
- ✅ Mixed Time Grid
- ✅ DS50 Time Type
- ✅ ECI Frame 
- ✅ J2K Frame
- ✅ Known Time Step UTC Time Type
- ✅ Known Time Step DS50 Time Type

### Single Propagate (api.v1.Propagator.Ephem)
- ✅ DS50 Time Type
- ✅ MSE Time Type
- ✅ UTC Time Type

### Info (api.v1.Propagator.Info)
- ✅ Info


