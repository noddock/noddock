package nodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

// DeepSeek Web Search 节点：调用 DeepSeek /v1/tools/search 网页搜索 API，
// 返回结构化搜索结果（title/url/snippet/summary/content/authority 等）。
// API key 走 vault ref（与 LLM 节点一致），endpoint 默认 DeepSeek 官方。
func init() {
	// Input
	param_query := graph.Parameter{
		ID:      "query",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}
	param_count := graph.Parameter{
		ID:      "count",
		Kind:    graph.ParamInt,
		Default: 0, // 0 = 不传，用 API 默认 10；有效范围 1-50
	}
	param_need_content := graph.Parameter{
		ID:      "need_content",
		Kind:    graph.ParamBool,
		Default: false, // false = 不抓正文（content 为空）
	}
	param_ori_question := graph.Parameter{
		ID:      "ori_question",
		Kind:    graph.ParamString,
		Default: "", // 可选：触发搜索的原始问题，提升查询理解
	}
	param_api := graph.Parameter{
		ID:      "api",
		Kind:    graph.ParamString,
		Default: "", // vault ref（解析为实际 API key）
	}
	param_endpoint := graph.Parameter{
		ID:      "endpoint",
		Kind:    graph.ParamString,
		Default: "https://api.deepseek.com/v1/tools/search",
	}
	// Output
	param_results := graph.Parameter{
		ID:      "results",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	param_total := graph.Parameter{
		ID:      "total",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "LLM--DeepSeek--WebSearch",
		Name:        "DeepSeek Web Search",
		Description: `Searches the web via DeepSeek /v1/tools/search and returns structured results. \n query: 1-128 chars, supports site:/-site:/quoted phrases. \n count: 1-50 (0 = API default 10). need_content: fetch page content. ori_question: original question (optional). \n api: vault ref for the DeepSeek API key. \n results: [{title,url,snippet,summary,content,site_name,date,authority,rerank_score}]; total: result count.`,
		Plist: []*graph.Parameter{
			&param_query,
			&param_count,
			&param_need_content,
			&param_ori_question,
			&param_api,
			&param_endpoint,
			&param_results,
			&param_total,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 1. 读取输入
			query, _ := ctx.Get(param_query.ID).(string)
			count, _ := ctx.Get(param_count.ID).(int)
			needContent, _ := ctx.Get(param_need_content.ID).(bool)
			oriQuestion, _ := ctx.Get(param_ori_question.ID).(string)
			api, _ := ctx.Get(param_api.ID).(string)
			endpoint, _ := ctx.Get(param_endpoint.ID).(string)

			if query == "" {
				return fmt.Errorf("DeepSeek WebSearch: query is required")
			}
			if api == "" {
				return fmt.Errorf("DeepSeek WebSearch: api (vault ref) is required")
			}
			if resolved, err := nRT.VaultLookup(api); err == nil {
				api = resolved
			} else {
				return fmt.Errorf("DeepSeek WebSearch: failed to resolve api key: %w", err)
			}

			// 2. 构建请求
			reqBody := struct {
				Query       string `json:"query"`
				Count       int    `json:"count,omitempty"`
				OriQuestion string `json:"ori_question,omitempty"`
				NeedContent bool   `json:"need_content,omitempty"`
			}{
				Query:       query,
				Count:       count, // 0 被 omitempty 省略 → API 默认 10
				OriQuestion: oriQuestion,
				NeedContent: needContent, // false 被省略 → API 默认 false
			}
			data, err := json.Marshal(reqBody)
			if err != nil {
				return fmt.Errorf("DeepSeek WebSearch: failed to marshal request: %w", err)
			}

			// 3. 发送请求
			req, err := http.NewRequestWithContext(nRT.GetRtCtx(), "POST", endpoint, bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("DeepSeek WebSearch: failed to build request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+api)

			client := &http.Client{Timeout: 0} // 流式/长请求靠 ctx 取消
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("DeepSeek WebSearch: request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
				return fmt.Errorf("DeepSeek WebSearch: server returned %d: %s", resp.StatusCode, string(body))
			}

			// 4. 解析响应（results 原样透传，保留全部字段）
			var result struct {
				Results json.RawMessage `json:"results"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("DeepSeek WebSearch: failed to parse response: %w", err)
			}
			if len(result.Results) == 0 {
				result.Results = json.RawMessage(`[]`)
			}
			var items []json.RawMessage
			if err := json.Unmarshal(result.Results, &items); err != nil {
				return fmt.Errorf("DeepSeek WebSearch: results is not an array: %w", err)
			}

			// 5. 输出
			ctx.Set(param_results.ID, result.Results)
			ctx.Set(param_total.ID, len(items))
			return nil
		},
	})
}
