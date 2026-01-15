// Package dag provides generic DAG algorithms that work with any data structure.
package dag

// DetectCycleFn detects a cycle in a graph using a higher-order function.
// Returns the cycle path if found, nil otherwise.
//
// Parameters:
//   - allIDs: All node identifiers in the graph
//   - getDeps: Function that returns dependencies (children) for a given node
//
// Example:
//
//	getDeps := func(id string) []string {
//	    switch id {
//	    case "A": return []string{"B", "C"}
//	    case "B": return []string{"C"}
//	    default: return nil
//	    }
//	}
//	cycle := DetectCycleFn([]string{"A", "B", "C"}, getDeps)
func DetectCycleFn[ID comparable](allIDs []ID, getDeps func(ID) []ID) []ID {
	visited := make(map[ID]bool)
	recStack := make(map[ID]bool)
	path := make([]ID, 0)

	var dfs func(id ID) []ID
	dfs = func(id ID) []ID {
		if recStack[id] {
			// Found cycle - extract it from path
			for i, pid := range path {
				if pid == id {
					cycle := make([]ID, len(path)-i)
					copy(cycle, path[i:])
					return cycle
				}
			}
			return path
		}

		if visited[id] {
			return nil
		}

		visited[id] = true
		recStack[id] = true
		path = append(path, id)

		for _, dep := range getDeps(id) {
			if cycle := dfs(dep); cycle != nil {
				return cycle
			}
		}

		path = path[:len(path)-1]
		recStack[id] = false
		return nil
	}

	for _, id := range allIDs {
		if cycle := dfs(id); cycle != nil {
			return cycle
		}
	}

	return nil
}

// HasCycleFn checks if a cycle exists in the graph.
// This is faster than DetectCycleFn when you don't need the cycle path.
func HasCycleFn[ID comparable](allIDs []ID, getDeps func(ID) []ID) bool {
	visited := make(map[ID]bool)
	recStack := make(map[ID]bool)

	var dfs func(id ID) bool
	dfs = func(id ID) bool {
		if recStack[id] {
			return true
		}
		if visited[id] {
			return false
		}

		visited[id] = true
		recStack[id] = true

		for _, dep := range getDeps(id) {
			if dfs(dep) {
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for _, id := range allIDs {
		if dfs(id) {
			return true
		}
	}

	return false
}

// CycleDetectable is an interface for types that can be checked for cycles.
type CycleDetectable[ID comparable] interface {
	ID() ID
	Dependencies() []ID
}

// DetectCycle detects a cycle in a collection of CycleDetectable items.
func DetectCycle[ID comparable, T CycleDetectable[ID]](items []T) []ID {
	// Build lookup map
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

	return DetectCycleFn(allIDs, getDeps)
}

// HasCycle checks if a cycle exists in a collection of CycleDetectable items.
func HasCycle[ID comparable, T CycleDetectable[ID]](items []T) bool {
	// Build lookup map
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

	return HasCycleFn(allIDs, getDeps)
}
