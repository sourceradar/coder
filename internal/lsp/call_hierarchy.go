package lsp

import (
	"context"
	"fmt"
	"time"

	"go.bug.st/lsp"
)

// PrepareCallHierarchy prepares call hierarchy items for a given position
func (m *Manager) PrepareCallHierarchy(filePath string, line, character int) ([]lsp.CallHierarchyItem, error) {
	// Ensure a server is running for this file
	language, err := m.ensureServerRunning(filePath)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	server := m.servers[language]
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert file path to URI
	fileURI := lsp.NewDocumentURI(filePath)

	// Create TextDocumentIdentifier
	textDocument := lsp.TextDocumentIdentifier{
		URI: fileURI,
	}

	// Create Position
	position := lsp.Position{
		Line:      line,
		Character: character,
	}

	// Request call hierarchy preparation
	params := &lsp.CallHierarchyPrepareParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: textDocument,
			Position:     position,
		},
	}

	items, rpcErr, err := server.Client.TextDocumentPrepareCallHierarchy(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare call hierarchy: %w", err)
	}

	if rpcErr != nil {
		return nil, fmt.Errorf("failed to prepare call hierarchy: %v", rpcErr)
	}

	return items, nil
}

// GetIncomingCalls gets all incoming calls for a call hierarchy item
func (m *Manager) GetIncomingCalls(item lsp.CallHierarchyItem) ([]lsp.CallHierarchyIncomingCall, error) {
	// Get language from item URI to find the appropriate server
	language, err := m.determineLanguageFromPath(item.URI.AsPath().Base())
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	server, exists := m.servers[language]
	m.mu.RUnlock()

	if !exists || !server.IsRunning {
		return nil, fmt.Errorf("no running server found for language: %s", language)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Request incoming calls
	params := &lsp.CallHierarchyIncomingCallsParams{
		Item: item,
	}

	calls, rpcErr, err := server.Client.CallHierarchyIncomingCalls(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to get incoming calls: %w", err)
	}

	if rpcErr != nil {
		return nil, fmt.Errorf("failed to get incoming calls: %v", rpcErr)
	}

	return calls, nil
}

// GetOutgoingCalls gets all outgoing calls from a call hierarchy item
func (m *Manager) GetOutgoingCalls(item lsp.CallHierarchyItem) ([]lsp.CallHierarchyOutgoingCall, error) {
	// Get language from item URI to find the appropriate server
	language, err := m.determineLanguageFromPath(item.URI.AsPath().String())
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	server, exists := m.servers[language]
	m.mu.RUnlock()

	if !exists || !server.IsRunning {
		return nil, fmt.Errorf("no running server found for language: %s", language)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Request outgoing calls
	params := &lsp.CallHierarchyOutgoingCallsParams{
		Item: item,
	}

	calls, rpcErr, err := server.Client.CallHierarchyOutgoingCalls(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing calls: %w", err)
	}

	if rpcErr != nil {
		return nil, fmt.Errorf("failed to get outgoing calls: %v", rpcErr)
	}

	return calls, nil
}
