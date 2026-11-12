package validate

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// MaxAliasExpansion bounds how much YAML aliases may add to a document once
// expanded, in bytes. Decoding shares an aliased value, but re-encoding the
// decoded config writes every alias out in full, so a small file could
// otherwise stand for gigabytes. Configs that reuse runners, images or scripts
// through anchors stay far below it.
const MaxAliasExpansion = 4 << 20

// aliasNodeCost is what each node reached through an alias, aliases included,
// counts for on top of its scalar value, so many small nodes are bounded as
// well as large ones, and so is the time spent counting them.
const aliasNodeCost = 32

// CheckAliasExpansion returns an error when the aliases in doc, a node tree
// from yaml.Unmarshal, expand to more than MaxAliasExpansion. It stops as soon
// as the budget is spent, so its cost is bounded whatever the document. An
// anchor that contains an alias to itself is left to the decoder to report.
func CheckAliasExpansion(doc *yaml.Node) error {
	c := aliasCounter{budget: MaxAliasExpansion, active: map[*yaml.Node]bool{}}
	if !c.walk(doc, false) {
		return fmt.Errorf("YAML aliases expand this document by more than %d MiB; use fewer or smaller aliases", MaxAliasExpansion>>20)
	}
	return nil
}

type aliasCounter struct {
	budget int
	// active holds the anchored nodes being expanded, to stop on a cycle.
	active map[*yaml.Node]bool
}

// walk reports whether n stays within the budget. Nodes count against it
// only when reached through an alias: the rest is bounded by the input.
func (c *aliasCounter) walk(n *yaml.Node, expanding bool) bool {
	if n == nil {
		return true
	}
	if expanding {
		c.budget -= len(n.Value) + aliasNodeCost
		if c.budget < 0 {
			return false
		}
	}
	if n.Kind == yaml.AliasNode {
		target := n.Alias
		if target == nil || c.active[target] {
			return true
		}
		c.active[target] = true
		ok := c.walk(target, true)
		delete(c.active, target)
		return ok
	}
	for _, child := range n.Content {
		if !c.walk(child, expanding) {
			return false
		}
	}
	return true
}
