package outline

import (
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"strings"
)

// Extract JavaScript outline directly from the code
func extractJSOutline(root *sitter.Node, content []byte) string {
	var result strings.Builder

	// Function to process a node and its children
	var processNode func(node *sitter.Node, indentLevel int)
	processNode = func(node *sitter.Node, indentLevel int) {
		indent := strings.Repeat(" ", indentLevel*2)

		// Process based on node type
		switch node.Kind() {
		case "program":
			// Process all children
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(uint(i))
				processNode(child, indentLevel)
			}

		case "function_declaration", "generator_function_declaration":
			// For JavaScript function declarations
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Get parameters
				paramNode := node.ChildByFieldName("parameters")
				paramText := ""
				if paramNode != nil {
					paramText = getNodeText(paramNode, content)
				}

				// Get documentation comment (JSDoc) if present
				doc := findDocComment(node, content, "javascript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write function declaration
				result.WriteString(fmt.Sprintf("%sfunction %s%s {\n", indent, name, paramText))
				result.WriteString(fmt.Sprintf("%s  // ...\n", indent))
				result.WriteString(fmt.Sprintf("%s}\n\n", indent))
			}

		case "method_definition":
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Skip private methods (those starting with #)
				if strings.HasPrefix(name, "#") {
					return
				}

				// Get parameters
				paramNode := node.ChildByFieldName("parameters")
				paramText := ""
				if paramNode != nil {
					paramText = getNodeText(paramNode, content)
				}

				// Check if it's a static method
				isStatic := false
				for j := 0; j < int(node.ChildCount()); j++ {
					if node.Child(uint(j)).Kind() == "static" {
						isStatic = true
						break
					}
				}

				prefix := ""
				if isStatic {
					prefix = "static "
				}

				// Get documentation comment if present
				doc := findDocComment(node, content, "javascript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write method definition
				result.WriteString(fmt.Sprintf("%s%s%s%s {\n", indent, prefix, name, paramText))
				result.WriteString(fmt.Sprintf("%s  // ...\n", indent))
				result.WriteString(fmt.Sprintf("%s}\n\n", indent))
			}

		case "class_declaration":
			// For JavaScript classes
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Get extends clause if any
				extendsNode := node.ChildByFieldName("superclass")
				extendsText := ""
				if extendsNode != nil {
					extendsText = " extends " + getNodeText(extendsNode, content)
				}

				// Get documentation comment if present
				doc := findDocComment(node, content, "javascript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write class declaration
				result.WriteString(fmt.Sprintf("%sclass %s%s {\n", indent, name, extendsText))

				// Process class body
				bodyNode := node.ChildByFieldName("body")
				if bodyNode != nil {
					for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
						child := bodyNode.NamedChild(uint(i))
						processNode(child, indentLevel+1)
					}
				}

				result.WriteString(fmt.Sprintf("%s}\n\n", indent))
			}

		case "export_statement":
			// Process exported declaration
			if node.NamedChildCount() > 0 {
				declaration := node.NamedChild(0)
				processNode(declaration, indentLevel)
			}

		case "lexical_declaration", "variable_declaration":
			// For variable declarations that might contain arrow functions
			if node.NamedChildCount() > 0 {
				for i := 0; i < int(node.NamedChildCount()); i++ {
					declarator := node.NamedChild(uint(i))
					if declarator.Kind() == "variable_declarator" && declarator.NamedChildCount() >= 2 {
						nameNode := declarator.NamedChild(0)
						valueNode := declarator.NamedChild(1)

						if valueNode.Kind() == "arrow_function" || valueNode.Kind() == "function" {
							name := getNodeText(nameNode, content)

							// Get declaration type
							declType := "var"
							if node.Kind() == "lexical_declaration" {
								if node.Child(0).Kind() == "let" {
									declType = "let"
								} else {
									declType = "const"
								}
							}

							// Get parameters
							paramNode := valueNode.ChildByFieldName("parameters")
							paramText := ""
							if paramNode != nil {
								paramText = getNodeText(paramNode, content)
							}

							// Get documentation comment if present
							doc := findDocComment(node, content, "javascript")
							if doc != "" {
								docLines := strings.Split(doc, "\n")
								for _, line := range docLines {
									result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
								}
							}

							// Write function
							if valueNode.Kind() == "arrow_function" {
								result.WriteString(fmt.Sprintf("%s%s %s = %s => {\n", indent, declType, name, paramText))
							} else {
								result.WriteString(fmt.Sprintf("%s%s %s = function%s {\n", indent, declType, name, paramText))
							}
							result.WriteString(fmt.Sprintf("%s  // ...\n", indent))
							result.WriteString(fmt.Sprintf("%s}\n\n", indent))
						}
					}
				}
			}
		}
	}

	processNode(root, 0)
	return result.String()
}
