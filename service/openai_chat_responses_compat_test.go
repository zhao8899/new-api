package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesRequestToChatCompletionsRequest(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Model:           "gpt-4.1",
		Instructions:    []byte(`"system prompt"`),
		Input:           []byte(`[{"role":"user","content":[{"type":"input_text","text":"hello"},{"type":"input_image","image_url":"https://example.com/a.png"}]}]`),
		Tools:           []byte(`[{"type":"function","function":{"name":"lookup","description":"find data","parameters":{"type":"object"}}}]`),
		ToolChoice:      []byte(`"auto"`),
		MaxOutputTokens: uintPtr(128),
		Temperature:     float64Ptr(0.2),
		TopP:            float64Ptr(0.9),
		Reasoning:       &dto.Reasoning{Effort: "high"},
		ServiceTier:     "flex",
		Store:           []byte(`false`),
		Stream:          boolPtr(false),
		StreamOptions:   &dto.StreamOptions{IncludeUsage: true},
	}

	got, err := ResponsesRequestToChatCompletionsRequest(req)
	require.NoError(t, err)
	require.Equal(t, "gpt-4.1", got.Model)
	require.Equal(t, "high", got.ReasoningEffort)
	require.Equal(t, uint(128), *got.MaxCompletionTokens)
	require.Equal(t, 0.2, *got.Temperature)
	require.Equal(t, 0.9, *got.TopP)
	require.False(t, *got.Stream)
	require.True(t, got.StreamOptions.IncludeUsage)
	require.Len(t, got.Tools, 1)
	require.Equal(t, "lookup", got.Tools[0].Function.Name)
	require.Equal(t, "auto", got.ToolChoice)
	var serviceTier string
	err = common.Unmarshal(got.ServiceTier, &serviceTier)
	require.NoError(t, err)
	require.Equal(t, "flex", serviceTier)
	require.Len(t, got.Messages, 2)
	require.Equal(t, "system", got.Messages[0].Role)
	require.Equal(t, "system prompt", got.Messages[0].StringContent())
	require.Equal(t, "user", got.Messages[1].Role)
	content := got.Messages[1].ParseContent()
	require.Len(t, content, 2)
	require.Equal(t, "hello", content[0].Text)
	require.Equal(t, "https://example.com/a.png", content[1].GetImageMedia().Url)
}

func uintPtr(v uint) *uint          { return &v }
func float64Ptr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool          { return &v }
