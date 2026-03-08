# OrderAssigner Test Plan

Run all tests from `Project/` directory:
```
go test -v -race ./OrderAssigner/...
```

---

## Unit Tests
Test individual helper functions in isolation — no executable needed.

| # | Function | What to test |
|---|----------|-------------|
| U1 | `hallrequestToBool` | `Acknowledged` → `true`, `NoCall`/`Exist` → `false` for all floors/dirs |
| U2 | `hallrequestToBool` | All-`NoCall` input → all-false output |
| U3 | `makeORAStateMap` | Node with valid ID → present in map with correct floor, direction, state strings |
| U4 | `makeORAStateMap` | Node with invalid ID (`""` or `"DISCONNECTED"`) → skipped, not in map |
| U5 | `makeORAStateMap` | `Acknowledged` cab req → `true`, others → `false` in the state map |
| U6 | `directionToString` | `MD_Up`→`"up"`, `MD_Down`→`"down"`, `MD_Stop`→`"stop"` |
| U7 | `stateToString` | `Moving`→`"moving"`, `DoorOpen`→`"doorOpen"`, `Idle`→`"idle"` |
| U8 | `insertCabCallsIntoOutput` | `Acknowledged` cab req at floor N → `output[N] = {true, true}` |
| U9 | `insertCabCallsIntoOutput` | ID not in node list → output unchanged |
| U10 | `insertCabCallsIntoOutput` | `NoCall` cab req → output[floor] not overwritten |

---

## Integration Tests
Test the full `Run()` pipeline end-to-end. Requires the `hall_request_assigner` executable.

| # | Scenario | Expected outcome |
|---|----------|-----------------|
| I1 | Happy path (existing `TestRunORA`) | Output received, no error |
| I2 | Acknowledged hall request in worldview | Own node or another node has that floor/dir set to `true` in output |
| I3 | `ownID = "one"`, acknowledged cab req at floor 0 for node "one" | `output[0] = {true, true}` |
| I4 | Send two worldviews in sequence | Two outputs received in order, no deadlock |
| I5 | Worldview where ownID is the only node | Output is non-empty and correct |

---

## Fault Tolerance Tests

| # | Scenario | Expected outcome |
|---|----------|-----------------|
| F1 | Empty worldview (`len(Nodes)==0`) | Skipped silently, no output sent, no crash |
| F2 | `ownID` not present in worldview nodes | Executable output has no entry for ownID → skipped with log, no output sent |
| F3 | Panic recovery — send worldview **with nodes**, outputCh pre-closed | Panic caught, `Run()` restarts, exits cleanly when wvCh closed |
| F4 | Executable timeout (takes > 2s) | `runORAExecutable` returns error, `Run()` continues without crash |
| F5 | Executable returns malformed JSON | `makeResult` returns error, `Run()` logs and continues |
| F6 | Invalid node ID in worldview | `makeORAStateMap` skips it, executable still runs on remaining nodes |

> **Note F3:** The worldview must have at least one node — otherwise the empty-worldview check short-circuits before the panic can be triggered.

---

## Concurrency Tests

| # | Scenario | Expected outcome |
|---|----------|-----------------|
| C1 | Send 5 worldviews rapidly to a size-1 buffered channel | Only latest is processed, 1 output received (intermediates dropped) |
| C2 | Send worldview, wait for output, repeat 3 times | Each worldview produces exactly one output, no deadlock |
| C3 | Close `wvCh` while `Run()` is mid-processing | `Run()` finishes current iteration, then exits cleanly |
