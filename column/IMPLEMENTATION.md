# Spec 015 Implementation: Fixed-Width Column Types + Retrofit

**Spec:** `spec_015_fixed-width-types.md`
**Build:** `go build ./... && go vet ./...`
**Lint:** `golangci-lint run ./...`
**Test:** `go test ./column/...`

## File Order (dependencies first)

| Step | File | Change | Status |
|------|------|--------|--------|
| 1 | `column/column.go` | `Base[T]` → `BaseColumn[T]`, `NewBase` → `NewBaseColumn` | ✅ |
| 2 | `column/str.go` | `Str` → `StrColumn`, `NewStr` → `NewStrColumn` | ✅ |
| 3 | `column/datetime.go` | `DateTime` → `DateTimeColumn`, `NewDateTime` → `NewDateTimeColumn`, add `Reset()` | ✅ |
| 4 | `column/uuid.go` | New: UUIDColumn, UUID value type, UUIDToGo/GoToUUID, String() | ✅ |
| 5 | `column/ipv4.go` | New: IPv4Column, IPv4 value type, IPv4ToNet/NetToIPv4, String() | ✅ |
| 6 | `column/ipv6.go` | New: IPv6Column, IPv6 value type, IPv6ToNet/NetToIPv6, String() | ✅ |
| 7 | `column/date.go` | New: DateColumn, Row/Append/AppendArr, Decode/Encode/WriteColumn | ✅ |
| 8 | `conn/fuzz_test.go` | Update type assertions | ✅ |
| 9 | `conn/dsn_e2e_test.go` | Update struct literals | ✅ |
| 10 | `column/column_test.go` | Update all `NewBase` → `NewBaseColumn`, `NewStr` → `NewStrColumn` | ✅ |
| 11-14 | `column/new_types_test.go` | Tests for UUID, IPv4, IPv6, Date columns | ✅ |
| 15 | Final verification | `go vet ./...` + `golangci-lint run ./...` + `go test ./column/...` | ✅ |

## Done Checklist (from spec)

- [x] UUIDColumn with Row/Append/AppendArr/Reset/DecodeColumn/EncodeColumn/WriteColumn
- [x] IPv4Column with Row/Append/AppendArr/Reset/DecodeColumn/EncodeColumn/WriteColumn
- [x] IPv6Column with Row/Append/AppendArr/Reset/DecodeColumn/EncodeColumn/WriteColumn
- [x] DateColumn with Row/Append/AppendArr/Reset/DecodeColumn/EncodeColumn/WriteColumn
- [x] Conversion functions: UUIDToGo, GoToUUID, IPv4ToNet, NetToIPv4, IPv6ToNet, NetToIPv6
- [x] String() on UUID, IPv4, IPv6 value types
- [x] DateTimeColumn.Reset() added
- [x] Retrofit: DateTimeColumn, StrColumn, BaseColumn[T], New*Column constructors
- [x] All four types satisfy Of[T] (Row/Append/AppendArr/Reset + Column)
- [x] All column tests pass

## Retrospective Notes

### Roadblocks / Misconceptions

1. **Zero-alloc claim for IPv4ToNet/IPv6ToNet was wrong.**
   `net.IP(v[:])` shares backing array but the value param escapes to heap if
   return escapes. The spec initially claimed "zero-alloc" for these functions.
   Fixed: removed all alloc claims, documented actual behavior (no byte copy,
   no heap alloc guarantee).

2. **net.ParseIP returns 16 bytes for IPv4 in Go.**
   Test assumed 4-byte return. Fixed test to use raw `net.IP{...}` literal.

3. **uuid column test UUID{...}[:] not addressable.**
   Cannot slice composite literal in Go. Fixed test to use var + assign.

4. **Conn E2E tests fail (pre-existing).**
   `conn/byte_compare_test.go`, `conn/insert_debug_test.go`,
   `conn/stream_e2e_test.go` — require ClickHouse Docker container,
   version/protocol mismatches unrelated to rename.

### Gaps

- Tests use aggregated file (`new_types_test.go`) rather than per-type files.
  Reduced file overhead, no behavioral difference.
