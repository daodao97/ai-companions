package websearch

import (
	"context"
	"time"
)

// SearchTool 搜索工具接口
type SearchTool interface {
	Search(ctx context.Context, query string) (*WebSearchResult, error)
}

// WebSearchResult 网络搜索结果
type WebSearchResult struct {
	Query     string       `json:"query"`
	Content   string       `json:"content"`
	Sources   []SourceInfo `json:"sources"`
	TaskID    string       `json:"task_id"`
	Timestamp time.Time    `json:"timestamp"`
}

// SourceInfo 信息源信息
type SourceInfo struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet"`
	Domain   string `json:"domain"`
	ShortURL string `json:"short_url,omitempty"`
}
