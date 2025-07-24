# JSON Performance Example

This example demonstrates the high-performance features of the Gravel JSON package.

## Features Demonstrated

1. **Basic Operations**: Standard marshal/unmarshal functionality
2. **Zero-allocation APIs**: `MarshalAppend` and `MarshalTo` for memory-efficient encoding
3. **Fast Parsing**: Simdjson-inspired two-stage parsing with tape architecture
4. **Zero-copy Search**: Extract values without allocating memory using `ZeroCopySearch`
5. **Performance Benchmarks**: Compare parsing and search performance

## Key Performance Features

- **SIMD Acceleration**: Uses SSE2/AVX2 instructions for faster string processing
- **Zero Allocations**: Memory-efficient marshaling and value extraction
- **Tape Architecture**: Two-stage parsing inspired by simdjson for maximum speed
- **Assembly Optimizations**: Hand-tuned AMD64 assembly for critical paths

## Running the Example

```bash
cd _examples/json-performance
go run main.go
```

## Expected Output

The example will show:
- Basic JSON operations
- Zero-allocation marshaling performance
- Fast parsing with tape inspection
- Zero-copy value extraction
- Performance benchmarks demonstrating speed improvements

## Performance Improvements

This JSON library provides significant performance improvements over standard approaches:
- 5x+ faster parsing through SIMD and tape architecture
- Zero-allocation marshaling reduces GC pressure
- Zero-copy search eliminates memory allocations during value extraction
