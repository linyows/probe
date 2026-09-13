# dag

Generic graph algorithms that work with any data structure.

## Installation

```go
import "github.com/linyows/probe/v2/dag"
```

## Algorithms

### Cycle Detection

Detects cycles in a graph using Depth-First Search (DFS) with a recursion stack.

| Function | Description | Complexity |
|----------|-------------|------------|
| `DetectCycleFn` | Detect cycle and return the cycle path | O(V + E) |

## Usage

Pass a list of IDs and a dependency getter function. Works with any data structure.

```go
// Dependencies: A → B → C → A
getDeps := func(id string) []string {
    switch id {
    case "A":
        return []string{"B"}
    case "B":
        return []string{"C"}
    case "C":
        return []string{"A"}
    default:
        return nil
    }
}
allIDs := []string{"A", "B", "C"}

if cycle := dag.DetectCycleFn(allIDs, getDeps); cycle != nil {
    fmt.Println("Cycle found:", cycle) // [A B C]
}
```
