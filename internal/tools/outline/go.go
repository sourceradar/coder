package outline

import (
	"fmt"
	"strings"

	"github.com/tree-sitter/go-tree-sitter"
)

// Extract Go symbols (kept for backwards compatibility)
func extractGoSymbols(root *tree_sitter.Node, content []byte) []SymbolInfo {
	// Legacy function, kept for compatibility
	var _ []SymbolInfo
	return nil
}

// Extract Go outline directly from the code
func extractGoOutline(root *tree_sitter.Node, content []byte) string {
	var result strings.Builder

	// Track methods by receiver type to organize them
	methodsByType := make(map[string][]string)

	// Function to process a node and its children
	var processNode func(node *tree_sitter.Node, indentLevel int)
	processNode = func(node *tree_sitter.Node, indentLevel int) {
		indent := strings.Repeat("\t", indentLevel)

		// Process based on node type
		switch node.Kind() {
		case "source_file":
			// Process all children
			var i uint
			for i = 0; i < node.NamedChildCount(); i++ {
				child := node.NamedChild(i)
				processNode(child, indentLevel)
			}

		case "function_declaration":
			// For Go functions
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Check if public (uppercase first letter in Go)
				isPublic := len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
				if !isPublic {
					return
				}

				// Get parameters and return type
				paramNode := node.ChildByFieldName("parameters")
				resultNode := node.ChildByFieldName("result")

				paramText := ""
				if paramNode != nil {
					paramText = getNodeText(paramNode, content)
				}

				resultText := ""
				if resultNode != nil {
					resultText = " " + getNodeText(resultNode, content)
				}

				// Get documentation comment if present
				doc := findDocComment(node, content, "go")
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
					}
				}

				// Write function declaration
				result.WriteString(fmt.Sprintf("%sfunc %s%s%s {\n", indent, name, paramText, resultText))
				result.WriteString(fmt.Sprintf("%s\t// ...\n", indent))
				result.WriteString(fmt.Sprintf("%s}\n\n", indent))
			}

		case "method_declaration":
			// For Go methods
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				// Check if public (uppercase first letter in Go)
				isPublic := len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
				if !isPublic {
					return
				}

				// Get receiver, parameters, and return type
				receiverNode := node.ChildByFieldName("receiver")
				paramNode := node.ChildByFieldName("parameters")
				resultNode := node.ChildByFieldName("result")

				receiverText := ""
				if receiverNode != nil {
					receiverText = getNodeText(receiverNode, content)
				}

				paramText := ""
				if paramNode != nil {
					paramText = getNodeText(paramNode, content)
				}

				resultText := ""
				if resultNode != nil {
					resultText = " " + getNodeText(resultNode, content)
				}

				// Extract receiver type to group methods
				receiverType := ""
				if receiverNode != nil && receiverNode.NamedChildCount() > 0 {
					for i := 0; i < int(receiverNode.NamedChildCount()); i++ {
						child := receiverNode.NamedChild(uint(i))
						if child.Kind() == "parameter_declaration" {
							typeNode := child.ChildByFieldName("type")
							if typeNode != nil {
								receiverType = strings.TrimPrefix(getNodeText(typeNode, content), "*")
								break
							}
						}
					}
				}

				// Get documentation comment if present
				doc := findDocComment(node, content, "go")
				docText := ""
				if doc != "" {
					docLines := strings.Split(doc, "\n")
					for _, line := range docLines {
						docText += fmt.Sprintf("\t%s// %s\n", indent, strings.TrimSpace(line))
					}
				}

				// Store method definition for later output with the appropriate type
				methodText := fmt.Sprintf("%s%sfunc %s %s%s%s {\n%s\t// ...\n%s}\n\n",
					docText, indent, receiverText, name, paramText, resultText, indent, indent)

				if receiverType != "" {
					methodsByType[receiverType] = append(methodsByType[receiverType], methodText)
				} else {
					// If we can't determine the receiver type, output it directly
					result.WriteString(methodText)
				}
			}

		case "type_declaration":
			// For Go types
			specNode := node.Child(1)
			if specNode != nil && specNode.Kind() == "type_spec" {
				nameNode := specNode.ChildByFieldName("name")
				if nameNode != nil {
					name := getNodeText(nameNode, content)

					// Check if public (uppercase first letter in Go)
					isPublic := len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
					if !isPublic {
						return
					}

					typeNode := specNode.ChildByFieldName("type")
					typeText := ""
					if typeNode != nil {
						typeText = getNodeText(typeNode, content)
					}

					// Get documentation comment if present
					doc := findDocComment(node, content, "go")
					if doc != "" {
						docLines := strings.Split(doc, "\n")
						for _, line := range docLines {
							result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
						}
					}

					// Check if it's a struct or interface type
					if typeNode != nil {
						if typeNode.Kind() == "struct_type" {
							// For struct types
							result.WriteString(fmt.Sprintf("%stype %s struct {\n", indent, name))

							// Parse struct fields
							if typeNode.NamedChildCount() > 0 {
								fieldsNode := typeNode.NamedChild(0)
								if fieldsNode != nil && fieldsNode.Kind() == "field_declaration_list" {
									for i := 0; i < int(fieldsNode.NamedChildCount()); i++ {
										fieldNode := fieldsNode.NamedChild(uint(i))
										if fieldNode.Kind() == "field_declaration" {
											fieldNameNode := fieldNode.ChildByFieldName("name")
											fieldTypeNode := fieldNode.ChildByFieldName("type")

											if fieldNameNode != nil && fieldTypeNode != nil {
												fieldName := getNodeText(fieldNameNode, content)
												fieldType := getNodeText(fieldTypeNode, content)

												// Check if field is public
												fieldIsPublic := len(fieldName) > 0 && fieldName[0] >= 'A' && fieldName[0] <= 'Z'
												if fieldIsPublic {
													result.WriteString(fmt.Sprintf("%s\t%s %s\n", indent, fieldName, fieldType))
												}
											}
										}
									}
								}
							}

							result.WriteString(fmt.Sprintf("%s}\n\n", indent))

							// Add methods for this type if any
							if methods, ok := methodsByType[name]; ok {
								for _, method := range methods {
									result.WriteString(method)
								}
							}
						} else if typeNode.Kind() == "interface_type" {
							// For interface types
							result.WriteString(fmt.Sprintf("%stype %s interface {\n", indent, name))

							// Parse interface methods
							if typeNode.NamedChildCount() > 0 {
								methodsNode := typeNode.NamedChild(0)
								if methodsNode != nil && methodsNode.Kind() == "method_spec_list" {
									for i := 0; i < int(methodsNode.NamedChildCount()); i++ {
										methodNode := methodsNode.NamedChild(uint(i))
										if methodNode.Kind() == "method_spec" {
											methodNameNode := methodNode.ChildByFieldName("name")
											methodParamsNode := methodNode.ChildByFieldName("parameters")
											methodResultNode := methodNode.ChildByFieldName("result")

											if methodNameNode != nil {
												methodName := getNodeText(methodNameNode, content)
												methodParams := ""
												if methodParamsNode != nil {
													methodParams = getNodeText(methodParamsNode, content)
												}

												methodResult := ""
												if methodResultNode != nil {
													methodResult = " " + getNodeText(methodResultNode, content)
												}

												result.WriteString(fmt.Sprintf("%s\t%s%s%s\n", indent, methodName, methodParams, methodResult))
											}
										}
									}
								}
							}

							result.WriteString(fmt.Sprintf("%s}\n\n", indent))
						} else {
							// For simple type aliases
							result.WriteString(fmt.Sprintf("%stype %s %s\n\n", indent, name, typeText))

							// Add methods for this type if any
							if methods, ok := methodsByType[name]; ok {
								for _, method := range methods {
									result.WriteString(method)
								}
							}
						}
					}
				}
			}

		case "const_declaration", "var_declaration":
			// For Go constants and variables
			isConst := node.Kind() == "const_declaration"
			declType := "var"
			if isConst {
				declType = "const"
			}

			// Get documentation comment if present
			doc := findDocComment(node, content, "go")
			if doc != "" {
				docLines := strings.Split(doc, "\n")
				for _, line := range docLines {
					result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
				}
			}

			result.WriteString(fmt.Sprintf("%s%s (\n", indent, declType))

			hasPublicItems := false
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(uint(i))
				if child.Kind() == "const_spec" || child.Kind() == "var_spec" {
					nameNode := child.ChildByFieldName("name")
					if nameNode != nil {
						name := getNodeText(nameNode, content)

						// Check if public (uppercase first letter in Go)
						isPublic := len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
						if !isPublic {
							continue
						}

						hasPublicItems = true

						typeNode := child.ChildByFieldName("type")
						valueNode := child.ChildByFieldName("value")

						typeText := ""
						if typeNode != nil {
							typeText = " " + getNodeText(typeNode, content)
						}

						valueText := ""
						if valueNode != nil {
							valueText = " = " + getNodeText(valueNode, content)
						}

						result.WriteString(fmt.Sprintf("%s\t%s%s%s\n", indent, name, typeText, valueText))
					}
				}
			}

			// Only output constants/variables block if it has public items
			if hasPublicItems {
				result.WriteString(fmt.Sprintf("%s)\n\n", indent))
			} else {
				// Remove the declaration if no public items
				result.Reset()
			}
		}
	}

	processNode(root, 0)
	return result.String()
}
