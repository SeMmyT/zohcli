package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// SchemaCmd outputs machine-readable command tree as JSON
type SchemaCmd struct {
	Command string `arg:"" optional:"" help:"Command path to show schema for (e.g., 'admin users')"`
}

// SchemaNode represents a node in the command tree
type SchemaNode struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"` // "application", "command", "argument"
	Help     string        `json:"help,omitempty"`
	Aliases  []string      `json:"aliases,omitempty"`
	Hidden   bool          `json:"hidden,omitempty"`
	Children []*SchemaNode `json:"commands,omitempty"`
	Flags    []*SchemaFlag `json:"flags,omitempty"`
	Args     []*SchemaArg  `json:"args,omitempty"`
}

// SchemaFlag represents a command flag
type SchemaFlag struct {
	Name     string   `json:"name"`
	Help     string   `json:"help,omitempty"`
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Default  string   `json:"default,omitempty"`
	Enum     []string `json:"enum,omitempty"`
	Short    string   `json:"short,omitempty"`
	Env      string   `json:"env,omitempty"`
}

// SchemaArg represents a positional argument
type SchemaArg struct {
	Name     string `json:"name"`
	Help     string `json:"help,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// Run executes the schema command
func (cmd *SchemaCmd) Run(ctx *kong.Context) error {
	rootNode := ctx.Model.Node

	var targetNode *kong.Node
	if cmd.Command == "" {
		targetNode = rootNode
	} else {
		var err error
		targetNode, err = findNodeByPath(rootNode, cmd.Command)
		if err != nil {
			return err
		}
	}

	schema := buildSchemaNode(targetNode)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(schema)
}

// buildSchemaNode recursively builds schema from Kong node
func buildSchemaNode(node *kong.Node) *SchemaNode {
	schema := &SchemaNode{
		Name:    node.Name,
		Type:    nodeTypeString(node.Type),
		Help:    node.Help,
		Aliases: node.Aliases,
		Hidden:  node.Hidden,
	}

	// Extract flags (skip system flags like --help, --version)
	for _, flag := range node.Flags {
		if flag.Name == "help" || flag.Name == "version" {
			continue
		}

		// Get type string from reflection
		typeName := "string"
		if flag.Value != nil && flag.Value.Target.IsValid() {
			typeName = fmt.Sprintf("%T", flag.Value.Target.Interface())
		}

		// Get environment variable (use first one if multiple)
		env := ""
		if len(flag.Envs) > 0 {
			env = flag.Envs[0]
		}

		schemaFlag := &SchemaFlag{
			Name:     flag.Name,
			Help:     flag.Help,
			Type:     typeName,
			Required: flag.Required,
			Default:  flag.Default,
			Env:      env,
		}

		if flag.Short != 0 {
			schemaFlag.Short = string(flag.Short)
		}

		// Extract enum values if present (stored in tag)
		if flag.Enum != "" {
			schemaFlag.Enum = strings.Split(flag.Enum, ",")
		}

		schema.Flags = append(schema.Flags, schemaFlag)
	}

	// Extract positional arguments
	for _, arg := range node.Positional {
		schemaArg := &SchemaArg{
			Name:     arg.Name,
			Help:     arg.Help,
			Required: arg.Required,
		}
		schema.Args = append(schema.Args, schemaArg)
	}

	// Recurse into children (skip hidden unless they're the current target)
	for _, child := range node.Children {
		schema.Children = append(schema.Children, buildSchemaNode(child))
	}

	return schema
}

// findNodeByPath walks the node tree to find a specific command path
func findNodeByPath(root *kong.Node, path string) (*kong.Node, error) {
	parts := strings.Fields(path)
	current := root

	for _, part := range parts {
		found := false
		for _, child := range current.Children {
			if child.Name == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("command not found: %s", path)
		}
	}

	return current, nil
}

// nodeTypeString converts Kong node type to string
func nodeTypeString(t kong.NodeType) string {
	switch t {
	case kong.ApplicationNode:
		return "application"
	case kong.CommandNode:
		return "command"
	case kong.ArgumentNode:
		return "argument"
	default:
		return "unknown"
	}
}
