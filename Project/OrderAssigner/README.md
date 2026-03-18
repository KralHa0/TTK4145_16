# OrderAssigner

Wraps the HRA executable to compute optimal hall-request assignments based on the current systems worldview.

The binaries are borrowed from: https://github.com/TTK4145/Project-resources/tree/master/cost_fns

## ORA – Hall Request Assigner

### What it is

A pre-compiled external binary that runs an optimization algorithm to decide which elevator should handle each hall call. It reasigns all hallrequest each call. 

Two binaries are included:

- `hall_request_assigner`: Linux
- `hall_request_assigner.exe`: Windows

### Input format

```json
{
  "hallRequests": [
    [false, false],
    [true,  false],
    [false, true ],
    [false, false]
  ],
  "states": {
    "<nodeID>": {
      "behaviour":   "idle | moving | doorOpen",
      "floor":       0,
      "direction":   "up | down | stop",
      "cabRequests": [false, true, false, false]
    }
  }
}
```

- `hallRequests`: `NumFloors × 2`: index `[floor][0]` = up, `[floor][1]` = down
- `states`: one entry per active elevator; only nodes considered available are included

### Output format

```json
{
  "<nodeID>": [
    [false, false],
    [true,  false],
    [false, false],
    [false, false]
  ]
}
```

Each elevator gets `NumFloors × 2` slice of assigned orders. After the ORA runs, the own node's cab requests are inserted into its output before the result is forwarded.
