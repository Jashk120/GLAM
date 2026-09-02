package scenario

import "fmt"

// validateArenaReferences enforces relationships that JSON Schema cannot express:
// the start node and every transition must point to an existing node, and a
// completion node must be reachable from the start node.
func validateArenaReferences(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	arena, ok := raw.(map[string]interface{})
	if !ok {
		return []string{"arena must be an object"}
	}
	flow, ok := arena["flow"].(map[string]interface{})
	if !ok {
		return []string{"arena.flow must be an object"}
	}
	start, _ := flow["start"].(string)
	nodes, ok := flow["nodes"].(map[string]interface{})
	if !ok {
		return []string{"arena.flow.nodes must be an object"}
	}
	if _, exists := nodes[start]; !exists {
		return []string{fmt.Sprintf("arena.flow.start %q does not exist", start)}
	}

	nexts := make(map[string][]string, len(nodes))
	var errs []string
	for id, rawNode := range nodes {
		node, ok := rawNode.(map[string]interface{})
		if !ok {
			continue
		}
		nodeID, _ := node["id"].(string)
		if nodeID != id {
			errs = append(errs, fmt.Sprintf("arena node key %q does not match id %q", id, nodeID))
		}
		if next, ok := node["next"].(string); ok && next != "" {
			nexts[id] = append(nexts[id], next)
		}
		if feedback, ok := node["feedback"].(map[string]interface{}); ok {
			if correct, ok := feedback["correct"].(map[string]interface{}); ok {
				if next, ok := correct["next"].(string); ok && next != "" {
					nexts[id] = append(nexts[id], next)
				}
			}
		}
	}
	for id, list := range nexts {
		for _, next := range list {
			if _, exists := nodes[next]; !exists {
				errs = append(errs, fmt.Sprintf("arena node %q next target %q does not exist", id, next))
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}

	visited := map[string]bool{}
	queue := []string{start}
	completeReachable := false
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if visited[id] {
			continue
		}
		visited[id] = true
		node, _ := nodes[id].(map[string]interface{})
		if node["type"] == "complete" {
			completeReachable = true
		}
		queue = append(queue, nexts[id]...)
	}
	if !completeReachable {
		return []string{"arena.flow must have a reachable complete node"}
	}
	return nil
}
