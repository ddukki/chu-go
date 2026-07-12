# chu-go

A Go client for the ClickHouse native (TCP) protocol. Type-safe generic columns, separate `Exec`/`Insert`/`Select` methods, built-in connection pooling.

```
go get github.com/ddukki/chu-go
```

## Why

chu-go is inspired by **[chconn](https://github.com/vahid-sohrabloo/chconn)** — the first Go client to prove generic columns over ClickHouse native protocol. chconn showed that Go 1.18 generics could eliminate per-type column structs and that single-allocation column decode (one `make([]T, rows)` per column) is dramatically faster than per-element append (35 vs 6683 allocs on 100M UInt64 reads).

We wanted chconn's generic column API, but we also wanted the protocol reliability, fuzz testing, and active maintenance of **[ch-go](https://github.com/ClickHouse/ch-go)**. Rather than compromise on either, chu-go combines both:

- **Generic columns like chconn** — `Base[T]`, `Str`, `Nullable[T]`, `LowCardinality[T]`, Tuple2–Tuple12.
- **Protocol from ch-go** — ch-go's wire layer is battle-tested with fuzz, golden, and e2e protocol tests. We don't reimplement the protocol.
- **Safe decode** — one `make([]T, rows)` allocation per column, direct `ReadFull` into the backing array. `Data` is always valid — no reader-buffer expiry, no corruption.
- **Error returns, not panics** — overflow, bounds, and protocol violations are safe by construction.
- **Fuzz + e2e tests from day one** — protocol-level fuzz tests, ch-go cross-verification, testcontainers-based e2e.
- **Built-in pool** — puddle-based connection pool with health checks, dead replica detection, configurable concurrency.

Other clients for context:

- **[ch-go](https://github.com/ClickHouse/ch-go)** — wire-level primitives, one `Do(ctx, Query{})` method, concrete column types per wire format. Excellent protocol tests but verbose column API.
- **[clickhouse-go](https://github.com/ClickHouse/clickhouse-go)** — struct-tag mapping, query builder, ORM-like API. Convenient for row-oriented code, heavy when you need column-level control.
- **[chconn](https://github.com/vahid-sohrabloo/chconn)** — Generic native-protocol columns. Pioneered the column-oriented generics approach chu-go builds on.

If you want raw protocol access, use ch-go. If you want ORM-style struct mapping, use clickhouse-go. If you want generic columns over native protocol with active maintenance, use chu-go.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ddukki/chu-go"
    "github.com/ddukki/chu-go/column"
)

func main() {
    ctx := context.Background()

    c, err := chu.Connect(ctx, chu.Config{
        Addr:     "localhost:9000",
        Password: "",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    c.Exec(ctx, "CREATE TABLE test (id UInt64, name String) ENGINE = Memory")

    idCol := column.NewBase[uint64]("id")
    idCol.AppendArr([]uint64{1, 2, 3})
    nameCol := column.NewStr("name")
    nameCol.Append("foo"); nameCol.Append("bar"); nameCol.Append("baz")
    c.Insert(ctx, "INSERT INTO test (id, name) VALUES", idCol, nameCol)

    outID := column.NewBase[uint64]("id")
    outName := column.NewStr("name")
    n, _ := c.Select(ctx, "SELECT id, name FROM test ORDER BY id", outID, outName)
    fmt.Printf("%d rows\n%v %v\n", n, outID.Data, outName.Data)
    // 3 rows
    // [1 2 3] [foo bar baz]
}
```

## API

### Connect

```go
c, err := chu.Connect(ctx, chu.Config{Addr: "localhost:9000"})
```

Or via DSN:

```go
import "github.com/ddukki/chu-go/conn"

cfg, _ := conn.ParseDSN("clickhouse://user:pass@localhost:9000/mydb?dial_timeout=10s&compress=lz4")
c, err := chu.Connect(ctx, cfg)
```

`Config` fields with zero-value defaults:

| Field | Default |
|-------|---------|
| `Addr` | `127.0.0.1:9000` |
| `User` | `default` |
| `Password` | `""` |
| `Database` | `default` |
| `Compression` | disabled |
| `DialTimeout` | no timeout |
| `ReadTimeout` | no timeout |
| `WriteTimeout` | no timeout |
| `TLSConfig` | nil (plain TCP) |

#### DSN format

```
clickhouse://[user[:password]@]host[:port][,host2[:port]]...[/database][?param=value&...]
```

Known connection params consumed from query string:

| Param | Type | Example |
|-------|------|---------|
| `dial_timeout` | duration | `10s` |
| `compress` | string | `lz4`, `none` |
| `read_timeout` | duration | `30s` |
| `write_timeout` | duration | `30s` |
| `secure` | bool | `true` |

All other params are passed as ClickHouse settings. Use `pool.ParsePoolDSN` for multi-host pool configuration.

### Exec

Execute DDL/DML queries that return no result rows.

```go
err := c.Exec(ctx, "CREATE TABLE t (x UInt64) ENGINE = Memory")
```

### Insert

Insert rows via native protocol. Pass one `Column` per table column, **including the column names**.

```go
col := column.NewBase[uint64]("id")
col.Append(1); col.Append(2)
c.Insert(ctx, "INSERT INTO t (id) VALUES", col)
```

### Select

Read results into pre-allocated columns. Returns row count.

```go
col := column.NewBase[uint64]("id")
n, err := c.Select(ctx, "SELECT id FROM t", col)
```

### Callbacks

Observe server-side telemetry during any operation:

```go
c.OnProgress = func(p proto.Progress) {
    log.Printf("rows=%d bytes=%d", p.Rows, p.Bytes)
}
c.OnProfile = func(p proto.Profile) { /* ... */ }
c.OnProfileEvent = func(p proto.ProfileEvent) { /* ... */ }
c.OnLog = func(l proto.Log) { /* ... */ }
```

### SelectStream

Stream large result sets block by block.

```go
s, _ := c.SelectStream(ctx, "SELECT * FROM large_table")
s.Bind(idCol, nameCol)
for s.Next() {
    // Each Next() appends one block to bound columns
    // Access col.Data to get all rows accumulated so far
}
if err := s.Err(); err != nil {
    log.Fatal(err)
}
s.Close()
```

Cancel mid-stream:

```go
s, _ := c.SelectStream(ctx, "SELECT * FROM huge_table")
s.Bind(col)
for s.Next() {
    if someCondition {
        s.Cancel()  // sends cancel, drains remaining blocks
        break
    }
}
s.Close()
```

### InsertStream

Insert data in multiple blocks.

```go
s, _ := c.InsertStream(ctx, "INSERT INTO t (id, name) VALUES")
s.Bind(idCol, nameCol)

idCol.AppendArr([]uint64{1, 2, 3})
nameCol.AppendArr([]string{"a", "b", "c"})
s.Append()  // sends block

idCol.Data = idCol.Data[:0]
nameCol.Data = nameCol.Data[:0]
idCol.AppendArr([]uint64{4, 5})
nameCol.AppendArr([]string{"d", "e"})
s.Append()  // sends second block

s.Close()  // sends end-of-data, reads server response
```

### DSN-based pool config

```go
import "github.com/ddukki/chu-go/pool"

cfg, _ := pool.ParsePoolDSN("clickhouse://host1:9000,host2:9000/mydb?pool_max_conns=10&health_check_interval=30s")
p, _ := pool.New(ctx, cfg)
```

### Connection pool

```go
import "github.com/ddukki/chu-go/pool"

p, _ := pool.New(ctx, pool.PoolConfig{
    Config:              chu.Config{Addr: "localhost:9000"},
    MaxConns:            10,
    HealthCheckInterval: 30 * time.Second,
})
defer p.Close()

p.Exec(ctx, "SELECT 1")
p.Select(ctx, "SELECT id FROM t", col)
p.Insert(ctx, "INSERT INTO t VALUES", col)

ss, _ := p.SelectStream(ctx, "SELECT * FROM large")
ss.Bind(col); for ss.Next() { /* ... */ }; ss.Close()
is, _ := p.InsertStream(ctx, "INSERT INTO t VALUES")
is.Bind(col); is.Append(); is.Close()
```

## Column types

| Type | Go type | Constructor |
|------|---------|-------------|
| UInt8, UInt16, UInt32, UInt64 | `uint8, uint16, uint32, uint64` | `NewBase[T]("name")` |
| Int8, Int16, Int32, Int64 | `int8, int16, int32, int64` | `NewBase[T]("name")` |
| Float32, Float64 | `float32, float64` | `NewBase[T]("name")` |
| String | `string` | `NewStr("name")` |
| Nullable(T) | `(T, bool)` | `NewNullable[T](inner)` |
| LowCardinality(T) | `T` (deduplicated) | `NewLowCardinality[T](inner)` |
| Tuple(T1..T12) | `Tuple2Value[T1,T2]` etc. | `NewTuple2(col1, col2)` |

Missing types (open an issue or PR): Decimal, Date, DateTime, Array, Map, IPv4, IPv6, UUID, Enum, Geo types.

## Compared to ch-go

| | ch-go | chu-go |
|---|---|---|
| **Column API** | Per-type structs (`ColUInt64`, `ColStr`, ...) | Generics (`Base[T]`, `Str`, ...) |
| **Operation dispatch** | Single `Do(ctx, Query{})` | `Exec` / `Insert` / `Select` |
| **Connection pool** | Not included | `pool/` package (puddle-based) |
| **Tuple support** | Manual | `Tuple2`–`Tuple12` codegen |
| **Nullable** | `ColNullable` wrapper | `Nullable[T]` generic |
| **LowCardinality** | `ColLowCardinality` wrapper | `LowCardinality[T]` generic |
| **Error handling** | Panics on overflow | Returned errors, no panics |

## Compared to clickhouse-go

| | clickhouse-go | chu-go |
|---|---|---|
| **Protocol** | Native + HTTP | Native only |
| **Query style** | `conn.QueryRowContext("SELECT ?", args)` | `c.Exec("SELECT 1")` (raw SQL) |
| **Result mapping** | `Scan(&a, &b)` struct tags | Manual column extraction |
| **Type system** | Reflection + `Scan` | Generic types + unsafe decode |
| **Connection pool** | Built-in | Separate `pool/` package |
| **API surface** | Large (~30 packages) | Small (~4 packages) |

## Performance

chu-go leads or ties all major select benchmarks vs ch-go, clickhouse-go, and chconn v2. Full results at [chu-go-bench/BENCHMARKS.md](https://github.com/ddukki/chu-go-bench/blob/main/BENCHMARKS.md).

### Select benchmarks

All select tests read from a local ClickHouse instance (testcontainers, Docker Desktop). Data is pre-inserted before the timed loop. Each result is the fastest of multiple runs after warmup.

#### Wide — 52-column table

Schema: `id UInt64` + 50× `Float64` + `label String`. Tests decode throughput for wide tables — common in analytics and time-series workloads.

Rows are verified by ID range (`WHERE id BETWEEN 0 AND N-1`) and cross-referenced against seed data.

| Driver | 100K rows | 1M rows |
|--------|-----------|---------|
| **chu-go** | ~70ms | **573ms** |
| ch-go | 61ms | 653ms |
| clickhouse-go | 76ms | 844ms |
| chconn v2 | 64ms | 583ms |

chu-go leads at 1M rows. ch-go is faster at 100K (smaller dataset amplifies overhead differences), but chu-go's unsafe `[]T` decode and column reuse pull ahead as row count grows. clickhouse-go's reflection-based column binding is ~47% slower at 1M.

#### Nullable — Nullable(UInt64) + Nullable(String)

Tests Nullable column decode — relevant for any table with optional fields, sparse data, or migrations where columns may be null.

Both `Nulls` bitmap and inner column `Data` are decoded and verified.

| Driver | 100K rows | 1M rows |
|--------|-----------|---------|
| **chu-go** | **14ms** | **89ms** |
| ch-go | 26ms | 179ms |
| clickhouse-go | 43ms | 215ms |
| chconn v2 | 14ms | 201ms |

chu-go ties chconn at 100K and widens to 2.3× faster at 1M. The gap comes from chconn's per-element `Read` on the inner column vs chu-go's bulk `DecodeColumn` into the backing array. clickhouse-go allocates per-element `*T` pointers, driving GC overhead.

#### LowCardinality — cardinality 100

Schema: `tag LowCardinality(String)` populated from 100 distinct values across 100K/1M rows. Tests the common pattern of low-cardinality string columns (status codes, categories, tiers).

No expansion on decode — chu-go stores dict + keys and resolves on read.

| Driver | 100K rows | 1M rows |
|--------|-----------|---------|
| **chu-go** | **5ms** | **19ms** |
| ch-go | 11ms | 62ms |
| clickhouse-go | 18ms | 116ms |
| chconn v2 | 6ms | 24ms |

chu-go leads by 2–3×. The lazy decode preserves the wire-format dict + narrow keys and resolves `Row()` in O(1) without materialization. chconn materializes into the inner column on decode, adding overhead. clickhouse-go's generic interface dispatch drives 6× slower decode.

### Insert benchmarks

#### InsertNarrow — 4-column single-block insert

Schema: `id UInt64, ts DateTime, value Float64, label String`. All rows in one INSERT block. Tests pure insert throughput for narrow tables.

| Driver | 100K rows | 1M rows |
|--------|-----------|---------|
| ch-go | 130ms | 168ms |
| **chu-go** | 145ms | 170ms |
| clickhouse-go | 128ms | 188ms |
| chconn v2 | 149ms | 177ms |

All four drivers cluster within ~12% at 1M rows. Single-block inserts are network-throughput-bound — the wire format overhead dominates, not the driver. chu-go is within ~1% of ch-go at 1M.

#### Batch Insert — convergence at batch=500

Tests the same 4-column narrow table with 1000 total rows split across varying batch sizes. Shows how batch granularity affects throughput.

| Batch size | ch-go | chu-go | clickhouse-go | chconn v2 |
|-----------|-------|--------|---------------|-----------|
| 10 (100 inserts) | 6.07s | 6.07s | 6.07s | 10.18s |
| 50 (20 inserts) | 1.21s | 1.21s | 1.21s | 2.04s |
| 100 (10 inserts) | 0.61s | 0.61s | 0.61s | 1.02s |
| 500 (2 inserts) | 0.20s | 0.20s | 0.20s | 0.20s |

ch-go, chu-go, and clickhouse-go are identical at every batch size — they all use the same buffered-column encoding strategy. chconn v2 is 68% slower at batch=10, converging at batch=500.

**Why chconn is slower at small batches:** chconn sends column headers and column data as separate `WriteTo` calls on the raw `net.Conn` — at minimum 11+ TCP writes per Insert vs ~2 for the others. On Windows, each extra syscall adds ~40ms overhead. At batch=500 (2 inserts total) the overhead drops below measurement noise.

**Key takeaway:** Batch granularity matters more than driver choice above ~100 rows. Use batches ≥500 for maximum throughput regardless of driver.

### Best practices

#### Column Reuse

**Anti-pattern — allocating inside a loop:**
```go
// SLOW: each iteration allocates a fresh column + backing array
for i := 0; i < b.N; i++ {
    col := column.NewBase[uint64]("id")
    c.Select(ctx, query, col)
}
```

**Correct — allocate once, reset between iterations:**
```go
// FAST: capacity is reused across iterations
col := column.NewBase[uint64]("id")
for i := 0; i < b.N; i++ {
    col.Data = col.Data[:0]
    c.Select(ctx, query, col)
}
```

All column types reuse backing array capacity on `.Data = .Data[:0]`. A single decode loop running 100× over 1M rows allocates ~once and flattens to near-zero thereafter.

### String Columns

chu-go's `Str` stores all string data in a contiguous `[]byte` buffer. After decode, `Data[i]` is a string header pointing into that buffer — zero per-string allocation.

**Access through `Data` or `Row(i)`:**
```go
outName := column.NewStr("name")
c.Select(ctx, "SELECT name FROM t", outName)
for _, s := range outName.Data { /* ... */ }
// or equivalently:
for i := 0; i < outName.Len(); i++ { _ = outName.Row(i) }
```

**Writes via `Append` create independent string headers** (caller's allocation, not the internal buffer). This is a one-time cost per value and only affects inserts, not selects.

### LowCardinality Access

After decode, `LowCardinality` stores dict + keys without materializing into the inner column. `Row(i)` resolves `O(1)` from `dict[keys[i]]` — no allocation.

**Use `Row(i)` for random access — O(1), no materialization cost:**
```go
lc := column.NewLowCardinality(column.NewStr("tag"))
c.Select(ctx, query, lc)
for i := 0; i < lc.Len(); i++ {
    _ = lc.Row(i)  // resolves from dict, no alloc
}
```

**Use `Values.Data` when you need the full `[]string`:**
```go
lc := column.NewLowCardinality(column.NewStr("tag"))
c.Select(ctx, query, lc)
allTags := lc.Values.Data  // triggers materialization (dict → []string)
```

Materialization copies all values from dict + keys into the inner column. This allocates once and is cached — subsequent access is free. `EncodeColumn` and `WriteColumn` also trigger materialization automatically.

**Insert path unaffected:** `lc.Append(val)` writes directly to `Values`, never touches dict/keys.

### Nullable Column Reset

Nullable columns have two stateful fields: `Nulls` and the inner column's `Data`. Both must be reset between iterations.

```go
outVal := column.NewNullable(column.NewBase[uint64]("val"))
for i := 0; i < b.N; i++ {
    outVal.Nulls = outVal.Nulls[:0]
    outVal.Data.Data = outVal.Data.Data[:0]  // inner column Data
    c.Select(ctx, query, outVal)
}
```

For `Nullable(Str)`:
```go
outStr := column.NewNullable(column.NewStr("val"))
for i := 0; i < b.N; i++ {
    outStr.Nulls = outStr.Nulls[:0]
    outStr.Data.Data = outStr.Data.Data[:0]
    c.Select(ctx, query, outStr)
}
```

### Batch Insert

Use batches of ≥500 rows. Single-block inserts are network-throughput-bound, not driver-bound — all four Go drivers cluster within ~12% at 1M rows.

```go
col := column.NewBase[uint64]("id")
for _, batch := range chunk(rows, 500) {
    col.Data = col.Data[:0]
    for _, v := range batch {
        col.Append(v.ID)
    }
    c.Insert(ctx, "INSERT INTO t (id) VALUES", col)
}
```

## Design

- **Wraps ch-go wire primitives**, not a reimplementation. Uses `proto.Reader`, `proto.Writer`, `proto.Buffer` from ch-go for all wire encoding.
- **Column-oriented.** You build columns, not rows. Insert passes columns; Select fills columns.
- **State machine.** `Initial → Ready → Busy → Ready → Closed`. No concurrent queries per connection.
- **Streaming.** SelectStream pulls blocks via Next(); InsertStream pushes blocks via Append(). Both use Bind() for pre-bound columns.
- **Revision-gated.** Checks server revision for features (`FeatureCustomSerialization`, `FeatureBlockInfo`, etc.) at runtime.
- **No panics.** All errors returned. Overflow, bounds, and protocol violations are safe by construction.
- **No panics in library code.** Overflow and bounds violations return errors. Safe by construction.
