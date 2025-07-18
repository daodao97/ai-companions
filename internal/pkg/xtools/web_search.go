package xtools

import (
	"companions/internal/pkg/tools/websearch"
	"companions/internal/pkg/xllm"
	"context"
	"encoding/json"
	"os"
)

var TavilySearchAPIKey = os.Getenv("TAVILY_API_KEY")

type WebSearchReq struct {
	Query string `json:"query"`
}

type WebSearchTool struct {
	Schema xllm.Tool
}

func NewWebSearchTool() ToolInterface {
	return &WebSearchTool{
		Schema: xllm.Tool{
			Name:        "web_search",
			Description: "从互联网搜索信息",
			Parameters: []xllm.Parameter{
				{
					Name:        "query",
					Description: "搜索关键词",
					Type:        xllm.ParameterTypeString,
					Required:    true,
				},
			},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	searchTool := websearch.NewTavilySearchTool(TavilySearchAPIKey)
	searchResp, err := searchTool.Search(ctx, args["query"].(string))
	if err != nil {
		return "", err
	}

	jsonData, _ := json.Marshal(searchResp)
	return string(jsonData), nil
}

func (t *WebSearchTool) GetSchema() xllm.Tool {
	return t.Schema
}
