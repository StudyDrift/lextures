package gradingagent

import (
	"encoding/json"
	"strings"
)

// NodeTypeGroup is a reusable subgraph node. It carries its own nested graph plus
// auto-derived input/output ports. External edges connect to the group node using
// port ids as handles. Groups are flattened (recursively) before validation and
// execution, so the rest of the engine never sees them.
const NodeTypeGroup = "group"

// maxGroupExpansions bounds the flatten loop, guarding against cyclic/over-deep nesting.
const maxGroupExpansions = 500

// GroupPort maps a group boundary handle to an internal node handle. For an input
// port the internal handle is a target; for an output port it is a source.
type GroupPort struct {
	ID     string `json:"id"`
	Label  string `json:"label,omitempty"`
	NodeID string `json:"nodeId"`
	Handle string `json:"handle"`
}

// GroupData is the typed shape of a group node's Data map.
type GroupData struct {
	Label    string        `json:"label,omitempty"`
	Subgraph WorkflowGraph `json:"subgraph"`
	Inputs   []GroupPort   `json:"inputs"`
	Outputs  []GroupPort   `json:"outputs"`
}

func isGroupNodeType(nodeType string) bool { return nodeType == NodeTypeGroup }

func graphContainsGroup(g *WorkflowGraph) bool {
	if g == nil {
		return false
	}
	for _, n := range g.Nodes {
		if n.Type == NodeTypeGroup {
			return true
		}
	}
	return false
}

// parseGroupData decodes a group node's Data into a typed GroupData.
func parseGroupData(n WorkflowNode) (GroupData, error) {
	raw, err := json.Marshal(n.Data)
	if err != nil {
		return GroupData{}, ValidationError{Field: "node:" + n.ID, Message: "Invalid group data."}
	}
	var gd GroupData
	if err := json.Unmarshal(raw, &gd); err != nil {
		return GroupData{}, ValidationError{Field: "node:" + n.ID, Message: "Invalid group data."}
	}
	return gd, nil
}

// groupSubgraph returns a group node's nested graph (used for recursive feature detection).
func groupSubgraph(n WorkflowNode) (*WorkflowGraph, error) {
	gd, err := parseGroupData(n)
	if err != nil {
		return nil, err
	}
	sub := gd.Subgraph
	return &sub, nil
}

// validateGroupStructure checks that a group's ports reference existing internal nodes.
func validateGroupStructure(groupID string, gd GroupData) error {
	internal := make(map[string]struct{}, len(gd.Subgraph.Nodes))
	for _, n := range gd.Subgraph.Nodes {
		internal[n.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(gd.Inputs)+len(gd.Outputs))
	check := func(ports []GroupPort, kind string) error {
		for _, p := range ports {
			id := strings.TrimSpace(p.ID)
			if id == "" {
				return ValidationError{Field: "node:" + groupID, Message: "Group " + kind + " ports must have an id."}
			}
			if _, dup := seen[id]; dup {
				return ValidationError{Field: "node:" + groupID, Message: "Duplicate group port id."}
			}
			seen[id] = struct{}{}
			if strings.TrimSpace(p.NodeID) == "" || strings.TrimSpace(p.Handle) == "" {
				return ValidationError{Field: "node:" + groupID, Message: "Group " + kind + " ports must reference an internal node handle."}
			}
			if _, ok := internal[p.NodeID]; !ok {
				return ValidationError{Field: "node:" + groupID, Message: "Group " + kind + " port references a node outside the group."}
			}
		}
		return nil
	}
	if err := check(gd.Inputs, "input"); err != nil {
		return err
	}
	return check(gd.Outputs, "output")
}

// FlattenWorkflowGraph recursively inlines every group node, returning an equivalent
// flat graph with no group nodes. Returns the graph unchanged when it has no groups.
func FlattenWorkflowGraph(g *WorkflowGraph) (WorkflowGraph, error) {
	if g == nil {
		return WorkflowGraph{Version: WorkflowVersion}, nil
	}
	cur := WorkflowGraph{
		Version: g.Version,
		Nodes:   append([]WorkflowNode(nil), g.Nodes...),
		Edges:   append([]WorkflowEdge(nil), g.Edges...),
	}
	for i := 0; i < maxGroupExpansions; i++ {
		idx := -1
		for j, n := range cur.Nodes {
			if n.Type == NodeTypeGroup {
				idx = j
				break
			}
		}
		if idx < 0 {
			if cur.Version == 0 {
				cur.Version = WorkflowVersion
			}
			return cur, nil
		}
		expanded, err := expandGroupNode(cur, idx)
		if err != nil {
			return WorkflowGraph{}, err
		}
		cur = expanded
	}
	return WorkflowGraph{}, ValidationError{Field: "workflowGraph", Message: "Group nesting is too deep or cyclic."}
}

// expandGroupNode replaces the group node at index gi with its inlined subgraph,
// prefixing internal ids and rewiring boundary edges through the group's ports.
func expandGroupNode(g WorkflowGraph, gi int) (WorkflowGraph, error) {
	gnode := g.Nodes[gi]
	gd, err := parseGroupData(gnode)
	if err != nil {
		return WorkflowGraph{}, err
	}
	if err := validateGroupStructure(gnode.ID, gd); err != nil {
		return WorkflowGraph{}, err
	}
	prefix := gnode.ID + "/"

	nodes := make([]WorkflowNode, 0, len(g.Nodes)-1+len(gd.Subgraph.Nodes))
	for idx, n := range g.Nodes {
		if idx == gi {
			continue
		}
		nodes = append(nodes, n)
	}
	for _, m := range gd.Subgraph.Nodes {
		mm := m
		mm.ID = prefix + m.ID
		nodes = append(nodes, mm)
	}

	inByID := make(map[string]GroupPort, len(gd.Inputs))
	for _, p := range gd.Inputs {
		inByID[p.ID] = p
	}
	outByID := make(map[string]GroupPort, len(gd.Outputs))
	for _, p := range gd.Outputs {
		outByID[p.ID] = p
	}

	edges := make([]WorkflowEdge, 0, len(g.Edges)+len(gd.Subgraph.Edges))
	for _, e := range gd.Subgraph.Edges {
		ne := e
		ne.ID = prefix + e.ID
		ne.Source = prefix + e.Source
		ne.Target = prefix + e.Target
		edges = append(edges, ne)
	}
	for _, e := range g.Edges {
		srcIs := e.Source == gnode.ID
		tgtIs := e.Target == gnode.ID
		if !srcIs && !tgtIs {
			edges = append(edges, e)
			continue
		}
		ne := e
		ne.ID = prefix + "boundary/" + e.ID
		if srcIs {
			p, ok := outByID[strings.TrimSpace(e.SourceHandle)]
			if !ok {
				return WorkflowGraph{}, ValidationError{Field: "edge:" + e.ID, Message: "Edge references an unknown group output port."}
			}
			ne.Source = prefix + p.NodeID
			ne.SourceHandle = p.Handle
		}
		if tgtIs {
			p, ok := inByID[strings.TrimSpace(e.TargetHandle)]
			if !ok {
				return WorkflowGraph{}, ValidationError{Field: "edge:" + e.ID, Message: "Edge references an unknown group input port."}
			}
			ne.Target = prefix + p.NodeID
			ne.TargetHandle = p.Handle
		}
		edges = append(edges, ne)
	}

	return WorkflowGraph{Version: g.Version, Nodes: nodes, Edges: edges}, nil
}
