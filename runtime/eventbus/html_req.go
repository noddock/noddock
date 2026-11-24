package eventbus

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// 节点 → 前端 HTML 的阻塞 RPC 请求。
type HtmlReq struct {
	PID     string
	Payload json.RawMessage
}

// 前端 → 节点的响应。
type HtmlResp struct {
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type RequestBroker struct{}

var HtmlReqBus = RequestBroker{}

// 发送req到全部自定义HTML中，阻塞等待结果。
func (b *RequestBroker) Request(ctx context.Context, req HtmlReq) (HtmlResp, error) {
	return HtmlResp{
		ID:      uuid.New().String(),
		Payload: json.RawMessage(`{"result":"mock","note":"no frontend in mock runtime"}`),
	}, nil
}
