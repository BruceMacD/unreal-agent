package responsesapi

import (
	"errors"
	"fmt"

	"github.com/unreallabsai/unreal-agent/internal/openaiapi"
	"github.com/unreallabsai/unreal-agent/harness/llm"
)

	input, err := requestInput(request.Input)
	if err != nil {
		return nil, err
	}
	tools, err := requestTools(request.Tools)
	if err != nil {
		return nil, err
	}

	var model openaiapi.ModelIdsResponses
	if err := model.FromModelIdsResponses1(openaiapi.ModelIdsResponses1(request.Model.ID)); err != nil {
		return nil, fmt.Errorf("encode model: %w", err)
	}
	store := false
	include := []openaiapi.IncludeEnum{openaiapi.ReasoningEncryptedContent}
	params := openaiapi.CreateResponse{
		Model:   &model,
		Store:   &store,
		Include: &include,
		Input:   &input,
	}
	if request.Model.MaxOutputTokens != nil {
		maxOutputTokens := int(*request.Model.MaxOutputTokens)
		params.MaxOutputTokens = &maxOutputTokens
	}
	if len(tools) != 0 {
		params.Tools = &tools
	}
	if err != nil {
		return nil, fmt.Errorf("encode response request: %w", err)
	}
}

func requestInput(items []llm.Item) (openaiapi.InputParam, error) {
	converted := make(openaiapi.InputParam1, 0, len(items))
	for index, item := range items {
		input, err := requestInputItem(item)
		if err != nil {
			return openaiapi.InputParam{}, fmt.Errorf("input item %d: %w", index, err)
		}
		converted = append(converted, input)
	}

	var input openaiapi.InputParam
	if err := input.FromInputParam1(converted); err != nil {
		return openaiapi.InputParam{}, fmt.Errorf("encode input: %w", err)
	}
	return input, nil
}

func requestInputItem(source llm.Item) (openaiapi.InputItem, error) {
	var item openaiapi.InputItem
	if source.Type == llm.ItemMessage && source.ProviderID == "" {
		message, ok := source.Data.(llm.Message)
		if !ok {
			return item, fmt.Errorf("message item data must be llm.Message, got %T", source.Data)
		}
		var content openaiapi.EasyInputMessage_Content
		if err := content.FromEasyInputMessageContent0(message.Text); err != nil {
			return item, err
		}
		converted := openaiapi.EasyInputMessage{
			Content: content,
			Role:    openaiapi.EasyInputMessageRole(message.Role),
		}
		if message.Phase != "" {
			phase := openaiapi.MessagePhase(message.Phase)
			converted.Phase = &phase
		}
		if err := setUnion(&item, converted); err != nil {
			return item, err
		}
		return item, nil
	}

	converted, err := requestItem(source)
	if err != nil {
		return item, err
	}
	if err := setUnion(&item, converted); err != nil {
		return item, err
	}
	return item, nil
}

func requestItem(source llm.Item) (openaiapi.Item, error) {
	var item openaiapi.Item
	switch source.Type {
	case llm.ItemMessage:
		message, ok := source.Data.(llm.Message)
		if !ok {
			return item, fmt.Errorf("message item data must be llm.Message, got %T", source.Data)
		}
		if message.Role != llm.RoleAssistant {
			var content openaiapi.InputContent
			if err := setUnion(&content, openaiapi.InputTextContent{
				Text: message.Text,
				Type: openaiapi.InputTextContentTypeInputText,
			}); err != nil {
				return item, err
			}
			messageType := openaiapi.InputMessageTypeMessage
			converted := openaiapi.InputMessage{
				Content: []openaiapi.InputContent{content},
				Role:    openaiapi.InputMessageRole(message.Role),
				Type:    &messageType,
			}
			if err := setUnion(&item, converted); err != nil {
				return item, err
			}
			return item, nil
		}
		content, err := requestOutputMessageContent(message)
		if err != nil {
			return item, err
		}
		converted := openaiapi.OutputMessage{
			Content: content,
			Id:      source.ProviderID,
			Role:    openaiapi.OutputMessageRoleAssistant,
			Status:  openaiapi.OutputMessageStatusCompleted,
			Type:    openaiapi.OutputMessageTypeMessage,
		}
		if message.Phase != "" {
			phase := openaiapi.MessagePhase(message.Phase)
			converted.Phase = &phase
		}
		if err := setUnion(&item, converted); err != nil {
			return item, err
		}
	case llm.ItemToolCall:
		call, ok := source.Data.(llm.ToolCall)
		if !ok {
			return item, fmt.Errorf("tool_call item data must be llm.ToolCall, got %T", source.Data)
		}
		converted := openaiapi.FunctionToolCall{
			CallId:    call.CallID,
			Name:      call.Name,
			Type:      openaiapi.FunctionCall,
		}
		if source.ProviderID != "" {
			converted.Id = &source.ProviderID
		}
		if err := setUnion(&item, converted); err != nil {
			return item, err
		}
	case llm.ItemToolResult:
		output, ok := source.Data.(llm.ToolResult)
		if !ok {
			return item, fmt.Errorf("tool_result item data must be llm.ToolResult, got %T", source.Data)
		}
		var value openaiapi.FunctionCallOutputItemParam_Output
			return item, err
		}
		converted := openaiapi.FunctionCallOutputItemParam{
			CallId: &output.CallID,
			Output: value,
			Type:   openaiapi.FunctionCallOutputItemParamTypeFunctionCallOutput,
		}
		if err := setUnion(&item, converted); err != nil {
			return item, err
		}
	case llm.ItemReasoning:
		reasoning, ok := source.Data.(llm.Reasoning)
		if !ok {
			return item, fmt.Errorf("reasoning item data must be llm.Reasoning, got %T", source.Data)
		}
		if len(reasoning.Raw) == 0 {
			return item, errors.New("reasoning item must carry the provider item in Raw")
		}
		if err := setUnion(&item, reasoning.Raw); err != nil {
			return item, err
		}
	default:
		return item, fmt.Errorf("unsupported input item type %q", source.Type)
	}
	return item, nil
}

func requestOutputMessageContent(message llm.Message) ([]openaiapi.OutputMessageContent, error) {
	content := make([]openaiapi.OutputMessageContent, 0, 1)
	if message.Text != "" {
		var part openaiapi.OutputMessageContent
		if err := setUnion(&part, openaiapi.OutputTextContent{
			Annotations: []openaiapi.Annotation{},
			Logprobs:    []openaiapi.LogProb{},
			Text:        message.Text,
			Type:        openaiapi.OutputText,
		}); err != nil {
			return nil, err
		}
		content = append(content, part)
	}
	return content, nil
}

func requestTools(source []llm.Tool) (openaiapi.ToolsArray, error) {
	tools := make(openaiapi.ToolsArray, 0, len(source))
	for index, source := range source {
		tool, err := requestTool(source)
		if err != nil {
			return nil, fmt.Errorf("tool %d: %w", index, err)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func requestTool(source llm.Tool) (openaiapi.Tool, error) {
	var converted any
	switch source.Type {
	case llm.ToolFunction:
		parameters := source.Parameters
		// The wire requires the field. The harness validates tool calls itself, and provider
		strict := false
		function := openaiapi.FunctionTool{
			Name:       source.Name,
			Parameters: &parameters,
			Strict:     &strict,
			Type:       openaiapi.FunctionToolTypeFunction,
		}
		if source.Description != "" {
			description := source.Description
			function.Description = &description
		}
		converted = function
	case llm.ToolHosted:
		switch source.Name {
		case "web_search":
			converted = openaiapi.WebSearchTool{Type: openaiapi.WebSearch}
		default:
			return openaiapi.Tool{}, fmt.Errorf("unsupported hosted tool name %q", source.Name)
		}
	default:
		return openaiapi.Tool{}, fmt.Errorf("unsupported tool type %q", source.Type)
	}

	var tool openaiapi.Tool
	if err := setUnion(&tool, converted); err != nil {
		return openaiapi.Tool{}, err
	}
	return tool, nil
}

func setUnion(destination json.Unmarshaler, source any) error {
	if err != nil {
		return err
	}
	return destination.UnmarshalJSON(body)
}
