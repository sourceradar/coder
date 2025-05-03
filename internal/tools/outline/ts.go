package outline

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Extract TypeScript symbols (kept for backwards compatibility)
func extractTSSymbols(root *sitter.Node, content []byte) []SymbolInfo {
	// Legacy function, kept for compatibility
	var _ []SymbolInfo
	return nil
}

// Extract TypeScript outline directly from the code
func extractTSOutline(root *sitter.Node, content []byte) string {
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
			// For TypeScript functions
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Get parameters
				paramNode := node.ChildByFieldName("parameters")
				paramText := ""
				if paramNode != nil {
					paramText = getNodeText(paramNode, content)
				}

				// Get return type if any
				returnNode := node.ChildByFieldName("return_type")
				returnText := ""
				if returnNode != nil {
					returnText = ": " + getNodeText(returnNode, content)
				}

				// Get documentation comment if present
				doc := findDocComment(node, content, "typescript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write function declaration
				result.WriteString(fmt.Sprintf("%sfunction %s%s%s {\n", indent, name, paramText, returnText))
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

				// Get return type if any
				returnNode := node.ChildByFieldName("return_type")
				returnText := ""
				if returnNode != nil {
					returnText = ": " + getNodeText(returnNode, content)
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
				doc := findDocComment(node, content, "typescript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write method definition
				result.WriteString(fmt.Sprintf("%s%s%s%s%s {\n", indent, prefix, name, paramText, returnText))
				result.WriteString(fmt.Sprintf("%s  // ...\n", indent))
				result.WriteString(fmt.Sprintf("%s}\n\n", indent))
			}

		case "class_declaration":
			// For TypeScript classes
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Get extends clause if any
				extendsNode := node.ChildByFieldName("superclass")
				extendsText := ""
				if extendsNode != nil {
					extendsText = " extends " + getNodeText(extendsNode, content)
				}

				// Get implements clause if any
				implementsNode := node.ChildByFieldName("implements_clause")
				implementsText := ""
				if implementsNode != nil {
					implementsText = " " + getNodeText(implementsNode, content)
				}

				// Get documentation comment if present
				doc := findDocComment(node, content, "typescript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write class declaration
				result.WriteString(fmt.Sprintf("%sclass %s%s%s {\n", indent, name, extendsText, implementsText))

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

		case "interface_declaration":
			// For TypeScript interfaces
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Get extends clause if any
				extendsNode := node.ChildByFieldName("extends_clause")
				extendsText := ""
				if extendsNode != nil {
					extendsText = " " + getNodeText(extendsNode, content)
				}

				// Get documentation comment if present
				doc := findDocComment(node, content, "typescript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write interface declaration
				result.WriteString(fmt.Sprintf("%sinterface %s%s {\n", indent, name, extendsText))

				// Process interface body for property and method signatures
				bodyNode := node.ChildByFieldName("body")
				if bodyNode != nil {
					for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
						child := bodyNode.NamedChild(uint(i))

						if child.Kind() == "property_signature" {
							nameNode := child.ChildByFieldName("name")
							typeNode := child.ChildByFieldName("type")

							if nameNode != nil && typeNode != nil {
								propName := getNodeText(nameNode, content)
								propType := getNodeText(typeNode, content)

								// Check for optional marker
								optional := ""
								for j := 0; j < int(child.ChildCount()); j++ {
									if child.Child(uint(j)).Kind() == "?" {
										optional = "?"
										break
									}
								}

								// Get doc comment
								propDoc := findDocComment(child, content, "typescript")
								if propDoc != "" {
									result.WriteString(fmt.Sprintf("%s  // %s\n", indent, propDoc))
								}

								result.WriteString(fmt.Sprintf("%s  %s%s: %s;\n", indent, propName, optional, propType))
							}
						} else if child.Kind() == "method_signature" {
							nameNode := child.ChildByFieldName("name")
							paramNode := child.ChildByFieldName("parameters")
							returnNode := child.ChildByFieldName("return_type")

							if nameNode != nil {
								methodName := getNodeText(nameNode, content)

								paramText := ""
								if paramNode != nil {
									paramText = getNodeText(paramNode, content)
								}

								returnText := ""
								if returnNode != nil {
									returnText = ": " + getNodeText(returnNode, content)
								}

								// Get doc comment
								methodDoc := findDocComment(child, content, "typescript")
								if methodDoc != "" {
									result.WriteString(fmt.Sprintf("%s  // %s\n", indent, methodDoc))
								}

								result.WriteString(fmt.Sprintf("%s  %s%s%s;\n", indent, methodName, paramText, returnText))
							}
						}
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

							// Get return type if any
							returnNode := valueNode.ChildByFieldName("return_type")
							returnText := ""
							if returnNode != nil {
								returnText = ": " + getNodeText(returnNode, content)
							}

							// Get documentation comment if present
							doc := findDocComment(node, content, "typescript")
							if doc != "" {
								docLines := strings.Split(doc, "\n")
								for _, line := range docLines {
									result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
								}
							}

							// Write function
							if valueNode.Kind() == "arrow_function" {
								result.WriteString(fmt.Sprintf("%s%s %s = %s%s => {\n", indent, declType, name, paramText, returnText))
							} else {
								result.WriteString(fmt.Sprintf("%s%s %s = function%s%s {\n", indent, declType, name, paramText, returnText))
							}
							result.WriteString(fmt.Sprintf("%s  // ...\n", indent))
							result.WriteString(fmt.Sprintf("%s}\n\n", indent))
						}
					}
				}
			}

		case "type_alias_declaration":
			// For TypeScript type aliases
			nameNode := node.ChildByFieldName("name")
			typeNode := node.ChildByFieldName("value")

			if nameNode != nil && typeNode != nil {
				name := getNodeText(nameNode, content)
				typeValue := getNodeText(typeNode, content)

				// Get documentation comment if present
				doc := findDocComment(node, content, "typescript")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				result.WriteString(fmt.Sprintf("%stype %s = %s;\n\n", indent, name, typeValue))
			}
		}
	}

	processNode(root, 0)
	return result.String()
}
