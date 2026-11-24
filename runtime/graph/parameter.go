package graph

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ParamKind string

const (
	ParamString ParamKind = "string"
	ParamFloat  ParamKind = "float"
	ParamInt    ParamKind = "int"
	ParamBool   ParamKind = "bool"
	ParamJSON   ParamKind = "json"
	ParamStream ParamKind = "stream"
	ParamAny    ParamKind = "any"
	ParamLogic  ParamKind = "logic"
)

// 参数连接语义：input 上游写入 / edit 手动编辑 / output 节点写出 / hidden 不显示。
type ParamConnect string

const (
	ConnInput  ParamConnect = "input"
	ConnEdit   ParamConnect = "edit"
	ConnOutput ParamConnect = "output"
	ConnHidden ParamConnect = "hidden"
)

type Parameter struct {
	ID         string
	Name       string
	Kind       ParamKind
	Connect    []ParamConnect
	Default    any
	Selections any
	Order      int
}

// 按 Kind 把任意值转成规范 Go 值（string/int/float64/bool/json.RawMessage/*Stream），转换失败返回 error。
func (k ParamKind) Cast(v any) (any, error) {
	switch k {
	case ParamString:
		switch t := v.(type) {
		case string:
			return t, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	case ParamInt:
		switch t := v.(type) {
		case int:
			return t, nil
		case int64:
			return int(t), nil
		case float64:
			return int(t), nil
		case string:
			i, err := strconv.Atoi(strings.TrimSpace(t))
			if err != nil {
				return nil, fmt.Errorf("invalid int: %v", t)
			}
			return i, nil
		default:
			return nil, fmt.Errorf("invalid int type: %T", v)
		}
	case ParamFloat:
		switch t := v.(type) {
		case float64:
			return t, nil
		case int:
			return float64(t), nil
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
			if err != nil {
				return nil, fmt.Errorf("invalid float: %v", t)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("invalid float type: %T", v)
		}
	case ParamBool:
		switch t := v.(type) {
		case bool:
			return t, nil
		case string:
			b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(t)))
			if err != nil {
				return nil, fmt.Errorf("invalid bool: %v", t)
			}
			return b, nil
		default:
			return nil, fmt.Errorf("invalid bool type: %T", v)
		}
	case ParamJSON:
		switch t := v.(type) {
		case string:
			if !json.Valid([]byte(t)) {
				return nil, fmt.Errorf("invalid json string")
			}
			return json.RawMessage(t), nil
		case []byte:
			if !json.Valid(t) {
				return nil, fmt.Errorf("invalid json raw")
			}
			return json.RawMessage(t), nil
		case json.RawMessage:
			if !json.Valid(t) {
				return nil, fmt.Errorf("invalid json raw")
			}
			return json.RawMessage(t), nil
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			return json.RawMessage(b), nil
		}
	case ParamLogic, ParamAny:
		return v, nil
	case ParamStream:
		if s, ok := v.(*Stream[string]); ok {
			return s, nil
		}
		if s, ok := v.(*Stream[[]byte]); ok {
			return s, nil
		}
		return nil, fmt.Errorf("invalid Stream param")
	default:
		return nil, fmt.Errorf("unsupported param kind: %s", k)
	}
}

// 按 Kind 转换值并校验 Selections 里的选项都合法，供节点注册和外部赋值复用。
func (p *Parameter) ValidateAndCast(v any) (any, error) {
	val, err := p.Kind.Cast(v)
	if err != nil {
		return nil, fmt.Errorf("param %s: %w", p.ID, err)
	}
	if p.Selections != nil {
		if anys, ok := p.Selections.([]any); ok {
			for _, s := range anys {
				if _, err := p.Kind.Cast(s); err != nil {
					return nil, fmt.Errorf("selection %s: %w", p.ID, err)
				}
			}
		}
	}
	return val, nil
}
