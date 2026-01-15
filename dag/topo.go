package dag

import "errors"

// ErrCycleDetected is returned when a cycle is detected during topological sort.
var ErrCycleDetected = errors.New("cycle detected in graph")

// TopologicalSortFn performs a topological sort on a graph.
// Returns the nodes in topological order (dependencies before dependents).
//
// Parameters:
//   - allIDs: All node identifiers in the graph
//   - getDeps: Function that returns dependencies (children) for a given node
//
// Returns: Sorted slice of node IDs, or error if cycle detected
func TopologicalSortFn[ID comparable](allIDs []ID, getDeps func(ID) []ID) ([]ID, error) {
	// Check for cycles first
	if HasCycleFn(allIDs, getDeps) {
		return nil, ErrCycleDetected
	}

	// Build reverse dependency map (who depends on me)
	dependents := make(map[ID][]ID)
	inDegree := make(map[ID]int)

	for _, id := range allIDs {
		if _, exists := inDegree[id]; !exists {
			inDegree[id] = 0
		}
		for _, dep := range getDeps(id) {
			dependents[dep] = append(dependents[dep], id)
			inDegree[id]++ // id depends on dep
		}
	}

	// Find all nodes with no dependencies (in-degree 0)
	queue := make([]ID, 0)
	for _, id := range allIDs {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	// Kahn's algorithm
	result := make([]ID, 0, len(allIDs))
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for dependents
		for _, dependent := range dependents[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// If not all nodes processed, there's a cycle (shouldn't happen as we checked above)
	if len(result) != len(allIDs) {
		return nil, ErrCycleDetected
	}

	return result, nil
}

// TopologicalSort performs a topological sort on a collection of CycleDetectable items.
func TopologicalSort[ID comparable, T CycleDetectable[ID]](items []T) ([]ID, error) {
	itemMap := make(map[ID]T)
	allIDs := make([]ID, len(items))
	for i, item := range items {
		id := item.ID()
		itemMap[id] = item
		allIDs[i] = id
	}

	getDeps := func(id ID) []ID {
		if item, ok := itemMap[id]; ok {
			return item.Dependencies()
		}
		return nil
	}

	return TopologicalSortFn(allIDs, getDeps)
}

// ComputeDescendantsFn computes all descendants of a node.
// Useful for impact analysis.
func ComputeDescendantsFn[ID comparable](allIDs []ID, startID ID, getDeps func(ID) []ID) []ID {
	// Build reverse dependency map
	dependents := make(map[ID][]ID)
	for _, id := range allIDs {
		for _, dep := range getDeps(id) {
			dependents[dep] = append(dependents[dep], id)
		}
	}

	// BFS to find all descendants
	visited := make(map[ID]bool)
	queue := []ID{startID}
	visited[startID] = true

	result := make([]ID, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current != startID {
			result = append(result, current)
		}

		for _, dependent := range dependents[current] {
			if !visited[dependent] {
				visited[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}

	return result
}

// ComputeAncestorsFn computes all ancestors of a node.
func ComputeAncestorsFn[ID comparable](allIDs []ID, startID ID, getDeps func(ID) []ID) []ID {
	// BFS using getDeps directly (it gives us ancestors/dependencies)
	visited := make(map[ID]bool)
	queue := []ID{startID}
	visited[startID] = true

	result := make([]ID, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current != startID {
			result = append(result, current)
		}

		for _, dep := range getDeps(current) {
			if !visited[dep] {
				visited[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	return result
}
