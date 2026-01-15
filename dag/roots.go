package dag

// FindRootsFn finds all root nodes (nodes with no incoming edges).
//
// Parameters:
//   - allIDs: All node identifiers in the graph
//   - getDeps: Function that returns dependencies (children) for a given node
//
// Returns: Slice of root node IDs
func FindRootsFn[ID comparable](allIDs []ID, getDeps func(ID) []ID) []ID {
	// Build set of all nodes that are dependencies of something
	hasParent := make(map[ID]bool)

	for _, id := range allIDs {
		for _, dep := range getDeps(id) {
			hasParent[dep] = true
		}
	}

	// Roots are nodes that are not dependencies of anything
	roots := make([]ID, 0)
	for _, id := range allIDs {
		if !hasParent[id] {
			roots = append(roots, id)
		}
	}

	return roots
}

// FindLeavesFn finds all leaf nodes (nodes with no outgoing edges).
//
// Parameters:
//   - allIDs: All node identifiers in the graph
//   - getDeps: Function that returns dependencies (children) for a given node
//
// Returns: Slice of leaf node IDs
func FindLeavesFn[ID comparable](allIDs []ID, getDeps func(ID) []ID) []ID {
	leaves := make([]ID, 0)

	for _, id := range allIDs {
		deps := getDeps(id)
		if len(deps) == 0 {
			leaves = append(leaves, id)
		}
	}

	return leaves
}

// FindRoots finds all root nodes from a collection of items.
func FindRoots[ID comparable, T CycleDetectable[ID]](items []T) []ID {
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

	return FindRootsFn(allIDs, getDeps)
}

// FindLeaves finds all leaf nodes from a collection of items.
func FindLeaves[ID comparable, T CycleDetectable[ID]](items []T) []ID {
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

	return FindLeavesFn(allIDs, getDeps)
}
