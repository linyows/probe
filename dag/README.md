# dag

A generic DAG (Directed Acyclic Graph) algorithms package. Provides generic functions that work with any data structure.

## Installation

```go
import "github.com/linyows/probe/dag"
```

## Algorithms

All algorithms have O(V + E) time complexity where V is the number of vertices and E is the number of edges.

### Cycle Detection

Detects cycles in a graph using Depth-First Search (DFS) with a recursion stack.

| Function | Description | Complexity |
|----------|-------------|------------|
| `DetectCycleFn` | Detect cycle and return the cycle path | O(V + E) |
| `HasCycleFn` | Check if cycle exists (faster when path not needed) | O(V + E) |
| `DetectCycle` | Interface-based cycle detection | O(V + E) |
| `HasCycle` | Interface-based cycle check | O(V + E) |

### Root & Leaf Discovery

Finds entry points (roots) and exit points (leaves) in a graph.

| Function | Description | Complexity |
|----------|-------------|------------|
| `FindRootsFn` | Find root nodes (in-degree = 0) | O(V + E) |
| `FindLeavesFn` | Find leaf nodes (out-degree = 0) | O(V + E) |
| `FindRoots` | Interface-based root discovery | O(V + E) |
| `FindLeaves` | Interface-based leaf discovery | O(V + E) |

### Topological Sort

Sorts nodes so that dependencies come before dependents using Kahn's algorithm (BFS-based).

| Function | Description | Complexity |
|----------|-------------|------------|
| `TopologicalSortFn` | Sort nodes by dependency order | O(V + E) |
| `TopologicalSort` | Interface-based topological sort | O(V + E) |

### Ancestor & Descendant Discovery

Traverses the graph to find all related nodes using Breadth-First Search (BFS).

| Function | Description | Complexity |
|----------|-------------|------------|
| `ComputeDescendantsFn` | Get all descendants of a node | O(V + E) |
| `ComputeAncestorsFn` | Get all ancestors of a node | O(V + E) |

## Usage

There are two ways to use this package: function-based and interface-based.

### Function-based (Fn suffix)

Pass a list of IDs and a dependency getter function. Works with any data structure.

```go
// Dependencies: A → B → C
getDeps := func(id string) []string {
    switch id {
    case "A":
        return []string{"B"}
    case "B":
        return []string{"C"}
    default:
        return nil
    }
}
allIDs := []string{"A", "B", "C"}

// Cycle detection
if dag.HasCycleFn(allIDs, getDeps) {
    cycle := dag.DetectCycleFn(allIDs, getDeps)
    fmt.Println("Cycle found:", cycle)
}

// Topological sort
sorted, err := dag.TopologicalSortFn(allIDs, getDeps)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Sorted:", sorted) // [C, B, A]

// Root & leaf discovery
roots := dag.FindRootsFn(allIDs, getDeps)  // [A]
leaves := dag.FindLeavesFn(allIDs, getDeps) // [C]
```

### Interface-based

Implement the `CycleDetectable` interface for cleaner integration with your types.

```go
type CycleDetectable[ID comparable] interface {
    ID() ID
    Dependencies() []ID
}
```

```go
type Task struct {
    name string
    deps []string
}

func (t Task) ID() string             { return t.name }
func (t Task) Dependencies() []string { return t.deps }

tasks := []Task{
    {"build", []string{"compile"}},
    {"compile", []string{"parse"}},
    {"parse", nil},
}

sorted, err := dag.TopologicalSort(tasks)
// sorted: ["parse", "compile", "build"]
```

## Use Cases

This package is useful for any system that needs to manage dependencies.

- **Build Systems**: Resolve task dependencies and determine execution order
- **Package Managers**: Determine installation order of dependencies
- **Schedulers**: Determine job execution order
- **Static Analysis**: Detect circular references
- **Impact Analysis**: Identify affected scope of changes
