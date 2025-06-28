# Gravel HTTP Server Optimizations

This document summarizes all the key optimizations applied to the `gravel` HTTP server codebase for **zero allocations**, **performance**, and **clarity**.

---

## 1. **Header Storage: Zero-Allocation**

- **Header names** are stored as `[MaxRequestHeaders][64]byte` arrays (for responses) or as `[][ ]byte` slices (for requests).
- **Header name lengths** are tracked in a parallel `[MaxRequestHeaders]int` array (for responses).
- **Header values** are stored as slices: `[MaxRequestHeaders][]byte`.
- **No heap allocations** occur for header names in responses; all are stored in preallocated arrays.

---

## 2. **Header Name Lowercasing and Comparison**

- All header keys are lowercased (ASCII only) before storage and comparison.
- Lowercasing is done in-place using a stack buffer, avoiding allocations.
- Comparisons are done byte-by-byte for speed.

---

## 3. **SetHeader Optimization (Response)**

- Only scans up to `HeaderCount` (not the full array).
- Uses a stack buffer for lowercased key.
- Updates value if header exists, otherwise adds a new header.
- No `bytes.EqualFold` or allocations in the hot path.

---

## 4. **AddHeader Optimization (Response)**

- Uses the same preallocated arrays.
- Appends values for repeated headers using `append` (allocates only for values, not names).
- Uses `bytes.EqualFold` for case-insensitive matching.

---

## 5. **Response Reset**

- `Reset()` resets `Status`, `HeaderCount`, and `Body` for reuse.
- Ensures no stale data or memory leaks between requests.

---

## 6. **Write Optimization (Response)**

- Uses `strconv.AppendInt` with stack buffers for status and content-length (zero allocation).
- Writes headers and body directly from preallocated arrays.
- Only iterates up to `HeaderCount`.

---

## 7. **Request Header Lookup Optimization**

- **Lower-case the lookup key** into a preallocated stack buffer (`lowerKey`) in `Request.HeaderValue`.
- **Compare header names** using byte-by-byte comparison, no allocations.
- **Early exit** if header name is nil or length does not match.
- **Zero allocations** for header lookup.

**Example:**
```go
func (req *Request) HeaderValue(key []byte) ([]byte, bool) {
    if len(key) == 0 {
        return nil, false
    }
    if len(key) > len(req.lowerKey) {
        return nil, false // key too long
    }
    for i := range key {
        c := key[i]
        if c >= 'A' && c <= 'Z' {
            c += 'a' - 'A'
        }
        req.lowerKey[i] = c
    }
    lookup := req.lowerKey[:len(key)]

    for i := 0; i < MaxRequestHeaders; i++ {
        headerName := req.HeaderNameList[i]
        if headerName == nil {
            break
        }
        if len(headerName) != len(lookup) {
            continue
        }
        eq := true
        for j := range lookup {
            if headerName[j] != lookup[j] {
                eq = false
                break
            }
        }
        if eq {
            return req.HeaderValueList[i], true
        }
    }
    return nil, false
}
```

---

## 8. **Request Header Parsing Optimization**

- **Lower-case header names in-place** during parsing (ASCII only, zero alloc).
- **Store header names and values** as slices into the request buffer, no allocations.
- **Break parsing** on empty line or error.

**Example:**
```go
for i := range MaxRequestHeaders {
    b, _, err := br.ReadLine()
    if err != nil {
        return err
    }
    if len(b) == 0 {
        if i+1 < MaxRequestHeaders {
            req.HeaderNameList[i] = nil
            req.HeaderValueList[i] = nil
        }
        break
    }
    cn := bytes.IndexByte(b, ' ')
    if cn < 0 {
        return errors.New("cannot find http request header name")
    }
    name := b[:cn-1]
    for j := range name {
        if name[j] >= 'A' && name[j] <= 'Z' {
            name[j] += 'a' - 'A'
        }
    }
    req.HeaderNameList[i] = name
    req.HeaderValueList[i] = b[cn+1:]
}
```

---

## 9. **General Best Practices**

- **No per-request heap allocations** for headers or status line.
- **Preallocate** all buffers and arrays at struct initialization.
- **Documented ASCII-only lowercasing** for header names.
- **Preallocated constants** for common header keys/values are recommended.

---

## 10. **Worker Pool and Ring Buffer**

- Worker pool uses a fixed-size array for zero allocations.
- Ring buffer is also array-based, not slice-based, for zero allocations.

---

## 11. **No Per-Connection Goroutine Allocation**

- Workers are pre-spawned at startup.
- Connections are dispatched via a channel to avoid per-connection goroutine stack allocations.

---

## 12. **Summary Table**

| Area                | Optimization                     | Allocation? |
|---------------------|----------------------------------|-------------|
| Header names        | Fixed array, lower-case, tracked | No          |
| Header values       | Slice, append for multi-value    | Only for values |
| Status/length       | Stack buffer + AppendInt         | No          |
| Worker pool         | Fixed array                      | No          |
| Goroutines          | Pre-spawned                      | No per-conn |
| Lowercasing         | ASCII only, stack buffer         | No          |
| Request lookup      | Stack buffer, byte compare       | No          |
| Request parsing     | In-place lower-case, slice store | No          |

---

**Review this document before making changes to ensure you keep the code zero-allocation