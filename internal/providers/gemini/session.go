package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"gemini-web-to-api/internal/providers"

	"github.com/google/uuid"
)

// ChatSession implements providers.ChatSession for Gemini
type ChatSession struct {
	client   *Client
	model    string
	metadata *providers.SessionMetadata
	history  []providers.Message
}

// SendMessage sends a message in the chat session
func (s *ChatSession) SendMessage(ctx context.Context, message string, options ...providers.GenerateOption) (*providers.Response, error) {
	s.client.reqMu.Lock()
	defer s.client.reqMu.Unlock()
	
	if s.client.at == "" {
		return nil, fmt.Errorf("client not initialized")
	}

	config := &providers.GenerateConfig{}
	for _, opt := range options {
		opt(config)
	}

	if config.DeepResearch {
		return s.sendDeepResearchMessage(ctx, message, options...)
	}

	// Build conversation context
	inner := []interface{}{
		[]interface{}{message},
		nil,
		s.buildMetadata(),
	}

	innerJSON, _ := json.Marshal(inner)
	outer := []interface{}{nil, string(innerJSON)}
	outerJSON, _ := json.Marshal(outer)

	formData := map[string]string{
		"at":    s.client.at,
		"f.req": string(outerJSON),
	}

	resp, err := s.client.httpClient.R().
		SetFormData(formData).
		SetQueryParam("at", s.client.at).
		Post(EndpointGenerate)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("chat failed with status: %d", resp.StatusCode)
	}

	response, err := s.client.parseResponse(resp.String())
	if err != nil {
		return nil, err
	}

	// Update session metadata
	s.updateSessionMetadata(response)

	// Update history
	s.history = append(s.history, providers.Message{
		Role:    "user",
		Content: message,
	})
	s.history = append(s.history, providers.Message{
		Role:    "model",
		Content: response.Text,
	})

	return response, nil
}

func (s *ChatSession) sendDeepResearchMessage(ctx context.Context, message string, options ...providers.GenerateOption) (*providers.Response, error) {
	// Phase 1: Planning
	reqID := uuid.New().String()

	// Index 0: prompt array
	promptArr := []interface{}{message, 0, nil, nil, nil, nil, 0}

	inner := make([]interface{}, 65)
	inner[0] = promptArr
	inner[1] = []interface{}{"en"}
	inner[2] = []interface{}{"", "", "", nil, nil, nil, nil, nil, nil, ""}
	inner[3] = "" 
	inner[4] = "7aed6d3c8dcea919033bfd7cdb523177"
	inner[17] = []interface{}{[]interface{}{0}} // Indicator for planning
	inner[54] = []interface{}{[]interface{}{[]interface{}{[]interface{}{[]interface{}{1}}}}} // Deep Research nested flag
	inner[55] = []interface{}{[]interface{}{1}}
	inner[59] = reqID

	innerJSON, _ := json.Marshal(inner)
	outer := []interface{}{nil, string(innerJSON)}
	outerJSON, _ := json.Marshal(outer)

	resp1, err := s.client.doRequest(ctx, outerJSON)
	if err != nil {
		return nil, err
	}

	res1, err := s.client.parseResponse(resp1.String())
	if err != nil {
		return nil, err
	}

	stateToken, ok := res1.Metadata["state_token"].(string)
	if !ok || stateToken == "" {
		return res1, nil
	}

	// Phase 2: Execution
	// Index 0: prompt array for execution
	promptArr2 := []interface{}{"Start research", 0, nil, nil, nil, nil, 0}

	inner2 := make([]interface{}, 65)
	inner2[0] = promptArr2
	inner2[1] = inner[1]
	inner2[2] = []interface{}{res1.Metadata["cid"], res1.Metadata["rid"], res1.Metadata["rcid"]}
	inner2[3] = stateToken
	inner2[17] = []interface{}{[]interface{}{1}} // Indicator for execution
	inner2[54] = inner[54]
	inner2[55] = inner[55]
	inner2[59] = reqID

	innerJSON2, _ := json.Marshal(inner2)
	outer2 := []interface{}{nil, string(innerJSON2), nil, stateToken}
	outerJSON2, _ := json.Marshal(outer2)

	resp2, err := s.client.doRequest(ctx, outerJSON2)
	if err != nil {
		return nil, err
	}

	response, err := s.client.parseResponse(resp2.String())
	if err != nil {
		return nil, err
	}

	// Update session metadata and history
	s.updateSessionMetadata(response)
	s.history = append(s.history, providers.Message{Role: "user", Content: message})
	s.history = append(s.history, providers.Message{Role: "model", Content: response.Text})

	return response, nil
}

func (s *ChatSession) updateSessionMetadata(response *providers.Response) {
	if response.Metadata != nil {
		if cid, ok := response.Metadata["cid"].(string); ok && cid != "" {
			if s.metadata == nil {
				s.metadata = &providers.SessionMetadata{}
			}
			s.metadata.ConversationID = cid
		}
		if rid, ok := response.Metadata["rid"].(string); ok && rid != "" {
			if s.metadata == nil {
				s.metadata = &providers.SessionMetadata{}
			}
			s.metadata.ResponseID = rid
		}
		if rcid, ok := response.Metadata["rcid"].(string); ok && rcid != "" {
			if s.metadata == nil {
				s.metadata = &providers.SessionMetadata{}
			}
			s.metadata.ChoiceID = rcid
		}
	}
}

// GetMetadata returns session metadata
func (s *ChatSession) GetMetadata() *providers.SessionMetadata {
	if s.metadata == nil {
		return &providers.SessionMetadata{
			Model: s.model,
		}
	}
	s.metadata.Model = s.model
	return s.metadata
}

// GetHistory returns conversation history
func (s *ChatSession) GetHistory() []providers.Message {
	return s.history
}

// Clear clears the conversation history
func (s *ChatSession) Clear() {
	s.history = []providers.Message{}
	s.metadata = nil
}

// buildMetadata builds metadata array for API request
func (s *ChatSession) buildMetadata() []interface{} {
	if s.metadata == nil {
		return []interface{}{nil, nil, nil}
	}

	return []interface{}{
		s.metadata.ConversationID,
		s.metadata.ResponseID,
		s.metadata.ChoiceID,
	}
}
