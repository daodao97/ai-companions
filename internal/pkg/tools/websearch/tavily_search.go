package websearch

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/daodao97/xgo/xjson"
	"github.com/daodao97/xgo/xlog"
	"github.com/daodao97/xgo/xrequest"
)

const (
	TavilySearchURL = "https://api.tavily.com/search"
)

// TavilySearchTool Tavily搜索工具
type TavilySearchTool struct {
	APIKey                   string
	SearchDepth              string // "basic" 或 "advanced"
	MaxResults               int
	UserAgent                string
	IncludeDomains           []string // 包含特定域名
	ExcludeDomains           []string // 排除特定域名
	IncludeAnswer            bool     // 包含答案
	IncludeRawContent        bool     // 包含原始内容
	IncludeImages            bool     // 包含图像
	IncludeImageDescriptions bool     // 包含图像描述
}

// NewTavilySearchTool 创建Tavily搜索工具
func NewTavilySearchTool(apiKey string) *TavilySearchTool {
	return &TavilySearchTool{
		APIKey:                   apiKey,
		SearchDepth:              "basic",
		MaxResults:               10,
		UserAgent:                "DeepResearch-Agent/1.0",
		IncludeDomains:           []string{},
		ExcludeDomains:           []string{},
		IncludeAnswer:            false,
		IncludeRawContent:        false,
		IncludeImages:            false,
		IncludeImageDescriptions: false,
	}
}

// NewTavilySearchToolWithOptions 创建带配置的Tavily搜索工具
func NewTavilySearchToolWithOptions(apiKey string, searchDepth string, maxResults int) *TavilySearchTool {
	if searchDepth == "" {
		searchDepth = "basic"
	}
	if maxResults <= 0 {
		maxResults = 10
	}

	return &TavilySearchTool{
		APIKey:                   apiKey,
		SearchDepth:              searchDepth,
		MaxResults:               maxResults,
		UserAgent:                "DeepResearch-Agent/1.0",
		IncludeDomains:           []string{},
		ExcludeDomains:           []string{},
		IncludeAnswer:            false,
		IncludeRawContent:        false,
		IncludeImages:            false,
		IncludeImageDescriptions: false,
	}
}

// TavilySearchOptions 高级配置选项
type TavilySearchOptions struct {
	SearchDepth              string   // 搜索深度
	MaxResults               int      // 最大结果数
	IncludeDomains           []string // 包含域名
	ExcludeDomains           []string // 排除域名
	IncludeAnswer            bool     // 包含答案
	IncludeRawContent        bool     // 包含原始内容
	IncludeImages            bool     // 包含图像
	IncludeImageDescriptions bool     // 包含图像描述
}

// NewTavilySearchToolWithAdvancedOptions 创建高级配置的Tavily搜索工具
func NewTavilySearchToolWithAdvancedOptions(apiKey string, options TavilySearchOptions) *TavilySearchTool {
	// 设置默认值
	if options.SearchDepth == "" {
		options.SearchDepth = "basic"
	}
	if options.MaxResults <= 0 {
		options.MaxResults = 10
	}
	if options.IncludeDomains == nil {
		options.IncludeDomains = []string{}
	}
	if options.ExcludeDomains == nil {
		options.ExcludeDomains = []string{}
	}

	return &TavilySearchTool{
		APIKey:                   apiKey,
		SearchDepth:              options.SearchDepth,
		MaxResults:               options.MaxResults,
		UserAgent:                "DeepResearch-Agent/1.0",
		IncludeDomains:           options.IncludeDomains,
		ExcludeDomains:           options.ExcludeDomains,
		IncludeAnswer:            options.IncludeAnswer,
		IncludeRawContent:        options.IncludeRawContent,
		IncludeImages:            options.IncludeImages,
		IncludeImageDescriptions: options.IncludeImageDescriptions,
	}
}

// tavilySearchRequest Tavily搜索请求结构
type tavilySearchRequest struct {
	APIKey                   string   `json:"api_key"`
	Query                    string   `json:"query"`
	SearchDepth              string   `json:"search_depth"`
	MaxResults               int      `json:"max_results"`
	IncludeDomains           []string `json:"include_domains"`
	ExcludeDomains           []string `json:"exclude_domains"`
	IncludeAnswer            bool     `json:"include_answer"`
	IncludeRawContent        bool     `json:"include_raw_content"`
	IncludeImages            bool     `json:"include_images"`
	IncludeImageDescriptions bool     `json:"include_image_descriptions"`
}

// tavilySearchResult Tavily页面搜索结果项
type tavilySearchResult struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent string  `json:"raw_content,omitempty"` // 原始内容
}

// tavilyImageResult Tavily图像搜索结果项
type tavilyImageResult struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// tavilySearchResponse Tavily搜索响应
type tavilySearchResponse struct {
	Query             string               `json:"query"`
	FollowUpQuestions []string             `json:"follow_up_questions,omitempty"`
	Answer            string               `json:"answer,omitempty"`
	Results           []tavilySearchResult `json:"results"`
	Images            []tavilyImageResult  `json:"images,omitempty"`
	ResponseTime      float64              `json:"response_time,omitempty"`
}

// TavilyResultItem 统一的结果项接口（页面或图像）
type TavilyResultItem struct {
	Type             string  `json:"type"`                        // "page" 或 "image"
	Title            string  `json:"title,omitempty"`             // 页面标题
	URL              string  `json:"url,omitempty"`               // 页面或图像URL
	Content          string  `json:"content,omitempty"`           // 页面内容
	Score            float64 `json:"score,omitempty"`             // 页面评分
	RawContent       string  `json:"raw_content,omitempty"`       // 原始内容
	ImageURL         string  `json:"image_url,omitempty"`         // 图像URL
	ImageDescription string  `json:"image_description,omitempty"` // 图像描述
}

// Search 实现SearchTool接口
func (t *TavilySearchTool) Search(ctx context.Context, query string) (*WebSearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("搜索查询词不能为空")
	}

	if t.APIKey == "" {
		return nil, fmt.Errorf("Tavily API密钥未配置")
	}

	xlog.Debug("开始Tavily搜索", xlog.String("query", query))

	// 构建请求
	reqBody := tavilySearchRequest{
		APIKey:                   t.APIKey,
		Query:                    query,
		SearchDepth:              t.SearchDepth,
		MaxResults:               t.MaxResults,
		IncludeDomains:           t.IncludeDomains,
		ExcludeDomains:           t.ExcludeDomains,
		IncludeAnswer:            t.IncludeAnswer,
		IncludeRawContent:        t.IncludeRawContent,
		IncludeImages:            t.IncludeImages,
		IncludeImageDescriptions: t.IncludeImageDescriptions,
	}

	// 发送HTTP请求
	resp, err := xrequest.New().
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", t.UserAgent).
		SetBody(reqBody).
		SetDebug(false).
		Post(TavilySearchURL)

	if err != nil {
		return nil, fmt.Errorf("Tavily搜索请求失败: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Tavily API返回错误状态码 %d: %s", resp.StatusCode(), resp.String())
	}

	// 解析响应
	var searchResponse tavilySearchResponse
	responseJson := resp.Json()

	// 解析基本信息
	searchResponse.Query = responseJson.Get("query").String()
	searchResponse.Answer = responseJson.Get("answer").String()
	searchResponse.ResponseTime = responseJson.Get("response_time").Float()

	// 解析页面结果
	_results := responseJson.Get("results").Array()
	for _, result := range _results {
		_result := xjson.New(result)
		searchResponse.Results = append(searchResponse.Results, tavilySearchResult{
			Title:      _result.Get("title").String(),
			URL:        _result.Get("url").String(),
			Content:    _result.Get("content").String(),
			Score:      _result.Get("score").Float(),
			RawContent: _result.Get("raw_content").String(),
		})
	}

	// 解析图像结果（如果包含）
	if t.IncludeImages {
		_images := responseJson.Get("images").Array()
		for _, image := range _images {
			_image := xjson.New(image)
			searchResponse.Images = append(searchResponse.Images, tavilyImageResult{
				URL:         _image.Get("url").String(),
				Description: _image.Get("description").String(),
			})
		}
	}

	// 解析后续问题（如果有）
	_followUps := responseJson.Get("follow_up_questions").Array()
	for _, followUp := range _followUps {
		if question := xjson.New(followUp).String(); question != "" {
			searchResponse.FollowUpQuestions = append(searchResponse.FollowUpQuestions, question)
		}
	}

	xlog.Debug("Tavily搜索完成",
		xlog.String("query", query),
		xlog.Int("page_results_count", len(searchResponse.Results)),
		xlog.Int("image_results_count", len(searchResponse.Images)),
		xlog.Float64("response_time", searchResponse.ResponseTime))

	// 转换为标准格式
	return t.convertToWebSearchResult(query, &searchResponse), nil
}

// convertToWebSearchResult 将Tavily搜索响应转换为标准WebSearchResult格式
func (t *TavilySearchTool) convertToWebSearchResult(query string, response *tavilySearchResponse) *WebSearchResult {
	// 转换信息源
	sources := make([]SourceInfo, 0, len(response.Results))
	var contentParts []string

	// 添加答案（如果有）
	if response.Answer != "" {
		answerPart := fmt.Sprintf("## 🎯 Tavily AI 答案\n\n%s\n", response.Answer)
		contentParts = append(contentParts, answerPart)
	}

	// 处理页面结果
	for i, result := range response.Results {
		// 提取域名
		domain := extractDomain(result.URL)

		source := SourceInfo{
			Title:   result.Title,
			URL:     result.URL,
			Snippet: result.Content,
			Domain:  domain,
		}
		sources = append(sources, source)

		// 构建内容，包含来源引用
		contentPart := fmt.Sprintf("### 📄 %s (评分: %.2f)\n\n%s\n\n**来源**: [%s](%s)\n**域名**: %s\n",
			result.Title, result.Score, result.Content, result.Title, result.URL, domain)

		// 如果包含原始内容，添加摘要
		if result.RawContent != "" && t.IncludeRawContent {
			rawContentSummary := truncateString(result.RawContent, 200)
			contentPart += fmt.Sprintf("**原始内容摘要**: %s\n", rawContentSummary)
		}

		contentParts = append(contentParts, contentPart)

		// 限制内容长度，避免过长
		if i >= 6 { // 最多显示前6个页面结果的详细内容
			break
		}
	}

	// 处理图像结果
	if t.IncludeImages && len(response.Images) > 0 {
		imageParts := []string{"## 🖼️ 相关图像"}
		for i, image := range response.Images {
			imagePart := fmt.Sprintf("![图像%d](%s)", i+1, image.URL)
			if image.Description != "" {
				imagePart += fmt.Sprintf("\n**描述**: %s", image.Description)
			}
			imagePart += fmt.Sprintf("\n**链接**: [查看原图](%s)", image.URL)
			imageParts = append(imageParts, imagePart)

			if i >= 4 { // 最多显示5张图像
				break
			}
		}
		contentParts = append(contentParts, strings.Join(imageParts, "\n\n"))
	}

	// 合并所有内容
	fullContent := fmt.Sprintf("# 🔍 搜索结果: %s\n\n%s", query, strings.Join(contentParts, "\n---\n\n"))

	// 添加统计信息
	stats := []string{}
	if len(response.Results) > 0 {
		stats = append(stats, fmt.Sprintf("📄 页面结果: %d个", len(response.Results)))
	}
	if len(response.Images) > 0 {
		stats = append(stats, fmt.Sprintf("🖼️ 图像结果: %d个", len(response.Images)))
	}
	if response.ResponseTime > 0 {
		stats = append(stats, fmt.Sprintf("⏱️ 响应时间: %.2f秒", response.ResponseTime))
	}

	if len(stats) > 0 {
		fullContent += fmt.Sprintf("\n\n---\n\n**搜索统计**: %s", strings.Join(stats, " | "))
	}

	// 添加后续问题（如果有）
	if len(response.FollowUpQuestions) > 0 {
		fullContent += "\n\n## 💡 建议的后续问题\n\n"
		for i, question := range response.FollowUpQuestions {
			fullContent += fmt.Sprintf("%d. %s\n", i+1, question)
		}
	}

	return &WebSearchResult{
		Query:     query,
		Content:   fullContent,
		Sources:   sources,
		Timestamp: time.Now(),
	}
}

// cleanResultsWithImages 清理并合并页面和图像结果
func (t *TavilySearchTool) cleanResultsWithImages(response *tavilySearchResponse) []TavilyResultItem {
	var cleanResults []TavilyResultItem

	// 添加页面结果
	for _, result := range response.Results {
		item := TavilyResultItem{
			Type:       "page",
			Title:      result.Title,
			URL:        result.URL,
			Content:    result.Content,
			Score:      result.Score,
			RawContent: result.RawContent,
		}
		cleanResults = append(cleanResults, item)
	}

	// 添加图像结果
	for _, image := range response.Images {
		item := TavilyResultItem{
			Type:             "image",
			ImageURL:         image.URL,
			ImageDescription: image.Description,
		}
		cleanResults = append(cleanResults, item)
	}

	return cleanResults
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractDomain 从URL中提取域名
func extractDomain(url string) string {
	// 简单的域名提取逻辑
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		domain := parts[0]
		// 移除端口号
		if colonIndex := strings.Index(domain, ":"); colonIndex != -1 {
			domain = domain[:colonIndex]
		}
		return domain
	}

	return "unknown"
}

// SetSearchDepth 设置搜索深度
func (t *TavilySearchTool) SetSearchDepth(depth string) {
	if depth == "basic" || depth == "advanced" {
		t.SearchDepth = depth
	}
}

// SetMaxResults 设置最大结果数
func (t *TavilySearchTool) SetMaxResults(maxResults int) {
	if maxResults > 0 && maxResults <= 50 { // Tavily限制最大50个结果
		t.MaxResults = maxResults
	}
}

// SetIncludeDomains 设置包含的域名
func (t *TavilySearchTool) SetIncludeDomains(domains []string) {
	t.IncludeDomains = domains
}

// SetExcludeDomains 设置排除的域名
func (t *TavilySearchTool) SetExcludeDomains(domains []string) {
	t.ExcludeDomains = domains
}

// EnableImages 启用图像搜索
func (t *TavilySearchTool) EnableImages(enableImageDescriptions bool) {
	t.IncludeImages = true
	t.IncludeImageDescriptions = enableImageDescriptions
}

// DisableImages 禁用图像搜索
func (t *TavilySearchTool) DisableImages() {
	t.IncludeImages = false
	t.IncludeImageDescriptions = false
}

// EnableAnswer 启用AI答案
func (t *TavilySearchTool) EnableAnswer() {
	t.IncludeAnswer = true
}

// EnableRawContent 启用原始内容
func (t *TavilySearchTool) EnableRawContent() {
	t.IncludeRawContent = true
}

// GetSearchStats 获取搜索统计信息
func (t *TavilySearchTool) GetSearchStats() map[string]interface{} {
	return map[string]interface{}{
		"provider":                   "Tavily",
		"search_depth":               t.SearchDepth,
		"max_results":                t.MaxResults,
		"api_configured":             t.APIKey != "",
		"include_domains":            len(t.IncludeDomains),
		"exclude_domains":            len(t.ExcludeDomains),
		"include_answer":             t.IncludeAnswer,
		"include_raw_content":        t.IncludeRawContent,
		"include_images":             t.IncludeImages,
		"include_image_descriptions": t.IncludeImageDescriptions,
	}
}
