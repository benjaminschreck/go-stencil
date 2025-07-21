# go-stencil Benchmarks

This document provides an overview of the benchmark suite and performance insights for go-stencil.

## Running Benchmarks

To run all benchmarks:
```bash
go test -bench=. -benchmem ./pkg/stencil
```

To run specific benchmarks:
```bash
go test -bench=BenchmarkRender -benchmem ./pkg/stencil
```

To run benchmarks with CPU profiling:
```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof ./pkg/stencil
go tool pprof cpu.prof
```

## Benchmark Categories

### 1. Template Preparation
- `BenchmarkPrepareTemplate_Simple`: Measures parsing of simple templates
- `BenchmarkPrepareTemplate_Complex`: Measures parsing of templates with loops, conditionals, and functions

### 2. Rendering Performance
- `BenchmarkRender_SimpleSubstitution`: Basic variable substitution
- `BenchmarkRender_Complex`: Complex template with nested data and functions
- `BenchmarkRender_LargeDataset`: Rendering with 100+ items in loops

### 3. Tokenizer Performance
- `BenchmarkTokenizer_Simple`: Basic template tokenization
- `BenchmarkTokenizer_Complex`: Full document tokenization
- `BenchmarkTokenizer_Nested`: Nested control structures

### 4. Expression Evaluation
- `BenchmarkExpressionParser_Simple`: Simple field access parsing
- `BenchmarkExpressionParser_Complex`: Mathematical expression parsing
- `BenchmarkExpressionEval_Simple`: Direct field evaluation
- `BenchmarkExpressionEval_Nested`: Nested object navigation
- `BenchmarkExpressionEval_Math`: Arithmetic operations
- `BenchmarkExpressionEval_FunctionCall`: Function execution

### 5. Control Structures
- `BenchmarkConditional_Simple`: If/else performance
- `BenchmarkLoop_Small`: Small loop iteration

### 6. Built-in Functions
- `BenchmarkFunction_StringOperations`: Chained string function calls

## Performance Insights

### Hot Paths Identified

1. **Tokenization** - The regex-based tokenizer is called frequently during template preparation
   - Optimization: Consider caching compiled regex patterns
   - Optimization: Use byte-based operations where possible

2. **Expression Evaluation** - Field access and expression evaluation happen for every template marker
   - Optimization: Cache parsed expressions within a template
   - Optimization: Use reflection-free path for common cases

3. **XML Processing** - Document parsing and reconstruction is memory-intensive
   - Optimization: Stream processing for large documents
   - Optimization: Reuse buffers for XML generation

4. **Loop Processing** - Loops with many iterations can be slow
   - Optimization: Pre-allocate slices when size is known
   - Optimization: Minimize DOM manipulations

### Memory Allocation Patterns

- Template preparation allocates memory for AST nodes
- Each render creates new document structure
- String concatenation in loops can cause many allocations

### Optimization Opportunities

1. **Template Caching** - Already implemented, ensure proper usage
2. **Expression Caching** - Cache parsed expressions by template
3. **Buffer Pooling** - Reuse buffers for XML generation
4. **Lazy Evaluation** - Defer expensive operations until needed
5. **Parallel Processing** - Process independent template sections concurrently

## Benchmark Results Format

When running benchmarks, results show:
- Operations per second (higher is better)
- Nanoseconds per operation (lower is better)
- Memory allocated per operation
- Number of allocations per operation

Example output:
```
BenchmarkRender_Simple-8    100000    10234 ns/op    2048 B/op    42 allocs/op
```

This means:
- 100,000 iterations were run
- Each operation took 10,234 nanoseconds
- Each operation allocated 2,048 bytes
- Each operation made 42 allocations

## Continuous Performance Monitoring

1. Run benchmarks before and after optimization attempts
2. Compare results using `benchstat` tool
3. Set performance regression thresholds
4. Include benchmark results in CI/CD pipeline

## Next Steps

1. Establish baseline performance metrics
2. Set performance goals for each operation type
3. Implement optimizations for identified hot paths
4. Create integration benchmarks for real-world scenarios