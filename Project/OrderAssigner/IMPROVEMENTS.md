# OrderAssigner — Code Improvement Plan

## Bugs (Correctness)

### 1. `insertCabCallsIntoOutput` — cab calls corrupt hall data
**File:** `orderAssigner.go:169`

Setting `[2]bool{true, true}` marks both hall-up and hall-down as active for a cab call.
This is intentional in the current design (signals FSM to stop regardless of direction),
but it should be documented clearly so the FSM implementer understands the convention.
If a separate cab-call slot is ever needed, this is the place to change.

---

### 2. Variable name shadows type name ✅ (already fixed)
~~`ORAInput := ORAInput{...}` in `worldviewToORAInput` — variable shadows the type.~~

---

## Missing / Broken Interface

### 3. Package-level `Run()` function is missing
**File:** `orderAssigner.go`

`main.go` calls `oa.Run(omToOraCh, oaToFsmCh, localID)` as a package-level function.
This does not exist — the code does not compile.

**Fix:** Add a thin wrapper:
```go
func Run(wvCh <-chan def.Worldview, outputCh chan<- def.AssignedOrders, ownID def.NodeID) {
    o := NewOrderAssigner(string(ownID), wvCh, outputCh)
    o.Run()
}
```

### 4. Constructor parameter type mismatch ✅
**File:** `orderAssigner.go:45`

`NewOrderAssigner` still takes `chan<- map[string][][2]bool` but the struct field is now
`chan<- def.AssignedOrders`. This prevents compilation.

**Fix:** Update the constructor parameter to `chan<- def.AssignedOrders`.

### 5. `Run()` method body sends wrong type ✅
**File:** `orderAssigner.go:117`

`o.outputCh <- output` tries to send a `map[string][][2]bool` to a `chan<- def.AssignedOrders`.
This does not compile.

**Fix:** Extract own ID's orders and send `def.AssignedOrders`:
```go
ownOrders, ok := output[o.ownID]
if !ok {
    fmt.Printf("OrderAssigner: own ID %q not found in HRA output\n", o.ownID)
    continue
}
var assigned def.AssignedOrders
for floor := range def.NumFloors {
    if floor < len(ownOrders) {
        assigned[floor] = ownOrders[floor]
    }
}
o.outputCh <- assigned
```

---

## Fault Tolerance 

### 6. No timeout on `exec.Command`✅
**File:** `orderAssigner.go:143` — **Priority: High**

If `hall_request_assigner` hangs, `Run()` blocks forever and the FSM receives no orders.

**Fix:** Use `exec.CommandContext` with a 2-second deadline:
```go
const oraTimeout = 2 * time.Second

func (o *OrderAssigner) runORAExecutable(jsonBytes []byte) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), oraTimeout)
    defer cancel()
    ret, err := exec.CommandContext(ctx, "../OrderAssigner/"+o.executable, "-i", string(jsonBytes)).CombinedOutput()
    ...
}
```
Requires adding `"context"` and `"time"` to imports.

### 7. No panic recovery in the goroutine
**File:** `orderAssigner.go` — **Priority: High**

A panic inside `Run()` (e.g. nil map, index out of bounds) crashes the entire program.
Since `Run()` is launched as a goroutine from `main.go`, there is no recovery.

**Fix:** Add `defer recover()` at the top of `Run()`:
```go
func (o *OrderAssigner) Run() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("OrderAssigner: panic recovered: %v\n", r)
        }
    }()
    ...
}
```

### 8. Buffer drain not implemented
**File:** `orderAssigner.go:87` — **Priority: High**

While the executable runs (~100ms+), multiple worldviews can queue up.
They are processed in FIFO order, meaning the FSM receives stale assignments.

**Fix:** Drain the channel after each receive, using only the latest worldview:
```go
for wv := range o.wvCh {
    for len(o.wvCh) > 0 {
        wv = <-o.wvCh
    }
    // process wv
}
```

### 9. No validation of empty worldview
**File:** `orderAssigner.go:87` — **Priority: Medium**

If `wv.Nodes` is empty, `makeORAStateMap` returns an empty map.
The executable may return an empty or unexpected result without any error.

**Fix:** Guard before processing:
```go
if len(wv.Nodes) == 0 {
    fmt.Println("OrderAssigner: skipping worldview with no nodes")
    continue
}
```

### 10. Disconnected nodes not filtered
**File:** `orderAssigner.go:197` — **Priority: Low**

Nodes with invalid IDs (e.g. `"DISCONNECTED"`, `""`) are included in the HRA state map,
which may cause unexpected executable behaviour.

**Fix:** Skip invalid nodes in `makeORAStateMap`:
```go
for _, node := range nodes {
    if !node.ID.IsValid() {
        continue
    }
    ...
}
```

### 11. Fragile relative path to executable
**File:** `orderAssigner.go:143` — **Priority: Medium**

`"../OrderAssigner/"` breaks if the process working directory differs from expected.

**Fix:** Resolve relative to the binary at startup using `os.Executable()`, or make the
path configurable via the constructor.

---

## Code Quality

### 12. Unstructured `fmt.Println` logging — **Priority: Low**

All debug output uses `fmt.Println`/`fmt.Printf`, making it impossible to filter or
silence without editing source code.

**Fix:** Replace with Go's `log` package:
```go
log.Println("OrderAssigner: received new worldview")
log.Printf("OrderAssigner: error: %v", err)
```

---

## Status Tracker

| # | Issue | Priority | Status |
|---|-------|----------|--------|
| 1 | `insertCabCallsIntoOutput` design — document convention | Low | Open |
| 2 | Variable shadowing in `worldviewToORAInput` | Medium | Done |
| 3 | Package-level `Run()` missing | Critical | Open |
| 4 | Constructor parameter type mismatch | Critical | Done |
| 5 | `Run()` body sends wrong type | Critical | Done |
| 6 | No timeout on `exec.Command` | High | Done |
| 7 | No panic recovery in goroutine | High | Open |
| 8 | Buffer drain not implemented | High | Open |
| 9 | No empty worldview guard | Medium | Open |
| 10 | Disconnected nodes not filtered | Low | Open |
| 11 | Fragile relative executable path | Medium | Open |
| 12 | `fmt.Println` logging | Low | Open |