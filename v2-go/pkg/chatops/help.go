package chatops

import (
	"fmt"
	"strings"
)

// CommandCategory groups related ChatOps commands.
type CommandCategory struct {
	Name     string
	Commands []CommandInfo
}

// CommandInfo describes a single ChatOps command.
type CommandInfo struct {
	Name        string
	Args        string
	Description string
}

// DefaultCommandCatalog is the unified command catalog shared across
// kudig and klaw ChatOps implementations.
var DefaultCommandCatalog = []CommandCategory{
	{
		Name: "Cluster",
		Commands: []CommandInfo{
			{Name: "cluster", Args: "", Description: "Show cluster overview (nodes, status, versions)"},
			{Name: "cluster", Args: "metrics", Description: "Get cluster metrics"},
			{Name: "cluster", Args: "chart", Description: "Send monitoring chart"},
		},
	},
	{
		Name: "Resources",
		Commands: []CommandInfo{
			{Name: "pod", Args: "[namespace]", Description: "List pods"},
			{Name: "deployment", Args: "[namespace]", Description: "List deployments"},
			{Name: "service", Args: "[namespace]", Description: "List services"},
			{Name: "node", Args: "", Description: "Show detailed node information"},
		},
	},
	{
		Name: "Operations",
		Commands: []CommandInfo{
			{Name: "scale", Args: "<ns> <deploy> <replicas>", Description: "Scale a deployment"},
			{Name: "restart", Args: "<ns> <deploy>", Description: "Restart a deployment"},
			{Name: "delete", Args: "<ns> <pod>", Description: "Delete a pod"},
			{Name: "logs", Args: "<ns> <pod>", Description: "Get pod logs"},
		},
	},
	{
		Name: "Analysis",
		Commands: []CommandInfo{
			{Name: "analyze", Args: "logs <ns> <pod>", Description: "Analyze pod logs"},
			{Name: "analyze", Args: "rbac", Description: "Analyze RBAC configuration"},
		},
	},
	{
		Name: "Monitoring",
		Commands: []CommandInfo{
			{Name: "monitor", Args: "status", Description: "Get monitoring status"},
			{Name: "monitor", Args: "alerts", Description: "Get active alerts"},
		},
	},
}

// RenderHelp renders the unified help message in Markdown format.
func RenderHelp(catalog []CommandCategory) string {
	var sb strings.Builder
	sb.WriteString("## ChatOps Commands\n\n")
	for _, cat := range catalog {
		sb.WriteString(fmt.Sprintf("### %s\n\n", cat.Name))
		sb.WriteString("| Command | Arguments | Description |\n")
		sb.WriteString("|---------|-----------|-------------|\n")
		for _, cmd := range cat.Commands {
			full := cmd.Name
			if cmd.Args != "" {
				full += " " + cmd.Args
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", cmd.Name, cmd.Args, cmd.Description))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
