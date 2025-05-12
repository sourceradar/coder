package outline

import (
	"fmt"
	"strings"

	"github.com/tree-sitter/go-tree-sitter"
)

func processNode(node *tree_sitter.Node, indentLevel int, content []byte, result *strings.Builder) {
	indent := strings.Repeat("\t", indentLevel)

	// Process based on node type
	switch node.Kind() {
	case "source_file":
		var i uint
		for i = 0; i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			processNode(child, indentLevel, content, result)
		}

	case "function_declaration":
		processFunction(node, content, result, indent)

	case "method_declaration":
		processMethod(node, content, result, indent)

	case "type_declaration":
		processType(node, content, result, indent)

	case "const_declaration", "var_declaration":
		processConstAndVar(node, content, result, indent)
	}
}

func processConstAndVar(node *tree_sitter.Node, content []byte, result *strings.Builder, indent string) {
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

	hasItems := false
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(uint(i))
		if child.Kind() == "const_spec" || child.Kind() == "var_spec" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				name := getNodeText(nameNode, content)

				hasItems = true

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

	// Only output block if it has items
	if hasItems {
		result.WriteString(fmt.Sprintf("%s)\n\n", indent))
	}
}

func processType(node *tree_sitter.Node, content []byte, result *strings.Builder, indent string) {
	specNode := node.Child(1)
	if specNode == nil || specNode.Kind() != "type_spec" {
		return
	}

	nameNode := specNode.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := getNodeText(nameNode, content)

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

	if typeNode == nil {
		return
	}

	if typeNode.Kind() == "struct_type" {
		processStruct(result, indent, name, typeNode, content)
	} else if typeNode.Kind() == "interface_type" {
		// For interface types
		processInterface(result, indent, name, typeNode, content)
		result.WriteString(fmt.Sprintf("%s}\n\n", indent))
	} else {
		result.WriteString(fmt.Sprintf("%stype %s %s\n\n", indent, name, typeText))
	}
}

func processInterface(result *strings.Builder, indent string, name string, typeNode *tree_sitter.Node, content []byte) {
	result.WriteString(fmt.Sprintf("%stype %s interface {\n", indent, name))

	// Parse interface methods
	if typeNode.NamedChildCount() == 0 {
		return
	}
	methodsNode := typeNode.NamedChild(0)
	if methodsNode == nil || methodsNode.Kind() != "method_spec_list" {
		return
	}
	for i := 0; i < int(methodsNode.NamedChildCount()); i++ {
		methodNode := methodsNode.NamedChild(uint(i))
		if methodNode.Kind() != "method_spec" {
			continue
		}
		methodNameNode := methodNode.ChildByFieldName("name")
		methodParamsNode := methodNode.ChildByFieldName("parameters")
		methodResultNode := methodNode.ChildByFieldName("result")

		if methodNameNode == nil {
			continue
		}
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

func processStruct(result *strings.Builder, indent string, name string, typeNode *tree_sitter.Node, content []byte) {
	// For struct types
	result.WriteString(fmt.Sprintf("%stype %s struct {\n", indent, name))
	defer result.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Parse struct fields
	if typeNode.NamedChildCount() == 0 {
		return
	}

	fieldsNode := typeNode.NamedChild(0)
	if fieldsNode == nil || fieldsNode.Kind() != "field_declaration_list" {
		return
	}

	for i := 0; i < int(fieldsNode.NamedChildCount()); i++ {
		fieldNode := fieldsNode.NamedChild(uint(i))
		if fieldNode.Kind() != "field_declaration" {
			continue
		}
		fieldNameNode := fieldNode.ChildByFieldName("name")
		fieldTypeNode := fieldNode.ChildByFieldName("type")

		// Handle both named fields and embedded fields
		if fieldTypeNode == nil {
			continue
		}
		if fieldNameNode != nil {
			// Regular field with name and type
			fieldName := getNodeText(fieldNameNode, content)
			fieldType := getNodeText(fieldTypeNode, content)
			result.WriteString(fmt.Sprintf("%s\t%s %s\n", indent, fieldName, fieldType))
		} else {
			// Embedded field (type only)
			embedType := getNodeText(fieldTypeNode, content)
			result.WriteString(fmt.Sprintf("%s\t%s\n", indent, embedType))
		}
	}
}

func processMethod(node *tree_sitter.Node, content []byte, result *strings.Builder, indent string) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, content)

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

	// Get documentation comment if present
	doc := findDocComment(node, content, "go")
	if doc != "" {
		docLines := strings.Split(doc, "\n")
		for _, line := range docLines {
			result.WriteString(fmt.Sprintf("%s// %s\n", indent, strings.TrimSpace(line)))
		}
	}

	// Write method declaration with dummy body
	result.WriteString(fmt.Sprintf("%sfunc %s %s%s%s { //... }\n\n",
		indent, receiverText, name, paramText, resultText))
}

func processFunction(node *tree_sitter.Node, content []byte, result *strings.Builder, indent string) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, content)

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

	// Write function declaration with dummy body
	result.WriteString(fmt.Sprintf("%sfunc %s%s%s { //... }\n\n", indent, name, paramText, resultText))
}

// Extract Go outline directly from the code
func extractGoOutline(root *tree_sitter.Node, content []byte) string {
	var result = new(strings.Builder)

	// Function to process a node and its children
	processNode(root, 0, content, result)

	outlineResult := result.String()
	if outlineResult == "" {
		// Add a fallback for empty result (for testing)
		outlineResult = "// No symbols found in the Go file\n"
		outlineResult += "// This might indicate a problem with the outline generation\n"
	}
	return outlineResult
}
