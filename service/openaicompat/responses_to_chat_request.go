package openaicompat

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func ResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Model == "" {
		return nil, errors.New("model is required")
	}

	out := &dto.GeneralOpenAIRequest{
		Model:                req.Model,
		Stream:               req.Stream,
		StreamOptions:        req.StreamOptions,
		MaxCompletionTokens:  req.MaxOutputTokens,
		Temperature:          req.Temperature,
		TopP:                 req.TopP,
		Metadata:             req.Metadata,
		Store:                req.Store,
		PromptCacheRetention: req.PromptCacheRetention,
	}
	if req.Reasoning != nil {
		out.ReasoningEffort = req.Reasoning.Effort
	}
	if req.ServiceTier != "" {
		serviceTier, err := common.Marshal(req.ServiceTier)
		if err != nil {
			return nil, err
		}
		out.ServiceTier = serviceTier
	}
	if len(req.Tools) > 0 {
		if err := common.Unmarshal(req.Tools, &out.Tools); err != nil {
			return nil, err
		}
	}
	if len(req.ToolChoice) > 0 {
		var toolChoice any
		if err := common.Unmarshal(req.ToolChoice, &toolChoice); err != nil {
			return nil, err
		}
		out.ToolChoice = toolChoice
	}

	messages := make([]dto.Message, 0)
	if len(req.Instructions) > 0 {
		var instructions string
		if common.GetJsonType(req.Instructions) == "string" {
			if err := common.Unmarshal(req.Instructions, &instructions); err != nil {
				return nil, err
			}
		} else {
			instructions = string(req.Instructions)
		}
		if instructions != "" {
			messages = append(messages, dto.Message{
				Role:    "system",
				Content: instructions,
			})
		}
	}

	inputMessages, err := responsesInputToMessages(req)
	if err != nil {
		return nil, err
	}
	messages = append(messages, inputMessages...)
	out.Messages = messages
	return out, nil
}

func responsesInputToMessages(req *dto.OpenAIResponsesRequest) ([]dto.Message, error) {
	if req == nil || len(req.Input) == 0 {
		return nil, nil
	}
	if common.GetJsonType(req.Input) == "string" {
		var text string
		if err := common.Unmarshal(req.Input, &text); err != nil {
			return nil, err
		}
		return []dto.Message{{Role: "user", Content: text}}, nil
	}
	if common.GetJsonType(req.Input) != "array" {
		return nil, nil
	}

	var inputs []dto.Input
	if err := common.Unmarshal(req.Input, &inputs); err != nil {
		return nil, err
	}

	messages := make([]dto.Message, 0, len(inputs))
	for _, input := range inputs {
		role := input.Role
		if role == "" {
			role = "user"
		}
		if common.GetJsonType(input.Content) == "string" {
			var text string
			if err := common.Unmarshal(input.Content, &text); err != nil {
				return nil, err
			}
			messages = append(messages, dto.Message{
				Role:    role,
				Content: text,
			})
			continue
		}
		if common.GetJsonType(input.Content) != "array" {
			continue
		}

		var items []map[string]any
		if err := common.Unmarshal(input.Content, &items); err != nil {
			return nil, err
		}
		parts := make([]dto.MediaContent, 0, len(items))
		for _, item := range items {
			switch common.Interface2String(item["type"]) {
			case "input_text", "output_text":
				parts = append(parts, dto.MediaContent{
					Type: dto.ContentTypeText,
					Text: common.Interface2String(item["text"]),
				})
			case "input_image":
				imageURL := common.Interface2String(item["image_url"])
				if imageURL == "" {
					if imageMap, ok := item["image_url"].(map[string]any); ok {
						imageURL = common.Interface2String(imageMap["url"])
					}
				}
				parts = append(parts, dto.MediaContent{
					Type: dto.ContentTypeImageURL,
					ImageUrl: &dto.MessageImageUrl{
						Url:    imageURL,
						Detail: common.Interface2String(item["detail"]),
					},
				})
			case "input_file":
				fileURL := common.Interface2String(item["file_url"])
				if fileURL == "" {
					if fileMap, ok := item["file_url"].(map[string]any); ok {
						fileURL = common.Interface2String(fileMap["url"])
					}
				}
				parts = append(parts, dto.MediaContent{
					Type: dto.ContentTypeFile,
					File: &dto.MessageFile{
						FileData: fileURL,
					},
				})
			}
		}
		if len(parts) == 0 {
			continue
		}
		message := dto.Message{Role: role}
		textOnly := true
		text := ""
		for _, part := range parts {
			if part.Type != dto.ContentTypeText {
				textOnly = false
				break
			}
			text += part.Text
		}
		if textOnly {
			message.SetStringContent(text)
		} else {
			message.SetMediaContent(parts)
		}
		messages = append(messages, message)
	}
	return messages, nil
}
