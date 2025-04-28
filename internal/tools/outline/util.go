package outline

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	"strings"
)

// Helper function to get node text from content
func getNodeText(node *sitter.Node, content []byte) string {
	return string(content[node.StartByte():node.EndByte()])
}

// Helper function to find documentation comment before a node
func findDocComment(node *sitter.Node, content []byte, language string) string {
	// Get the previous sibling
	if node.Parent() == nil {
		return ""
	}

	var comment string
	currentNode := node.PrevNamedSibling()

	for currentNode != nil {
		nodeType := currentNode.Kind()

		if strings.Contains(nodeType, "comment") {
			text := getNodeText(currentNode, content)
			// Clean up the comment text based on language
			switch language {
			case "go":
				text = strings.TrimPrefix(text, "//")
				text = strings.TrimPrefix(text, "/*")
				text = strings.TrimSuffix(text, "*/")
			case "javascript", "typescript":
				text = strings.TrimPrefix(text, "//")
				text = strings.TrimPrefix(text, "/*")
				text = strings.TrimSuffix(text, "*/")
				// Handle JSDoc comments
				text = strings.TrimPrefix(text, "*")
			case "python":
				text = strings.TrimPrefix(text, "#")
				// Handle docstrings
				text = strings.TrimPrefix(text, "\"\"\"")
				text = strings.TrimSuffix(text, "\"\"\"")
				text = strings.TrimPrefix(text, "'''")
				text = strings.TrimSuffix(text, "'''")
			}

			text = strings.TrimSpace(text)
			if comment == "" {
				comment = text
			} else {
				comment = text + "\n" + comment
			}

			currentNode = currentNode.PrevNamedSibling()
		} else {
			break
		}
	}

	return comment
}
