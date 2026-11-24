package tool

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	schemaCache = make(map[string]*jsonschema.Schema)
	cacheMu     sync.RWMutex
)

// 用 schema 内容的 SHA-256 当缓存 key（定长、URI 安全、格式无关）。节点多次调用时服用schema
func schemaKey(schemaJSON []byte) string {
	sum := sha256.Sum256(schemaJSON)
	return hex.EncodeToString(sum[:])
}

// 编译 JSON Schema，按内容哈希缓存编译结果避免重复编译。
func getSchema(schemaJSON []byte) (*jsonschema.Schema, error) {
	key := schemaKey(schemaJSON)
	cacheMu.RLock()
	if s, ok := schemaCache[key]; ok {
		cacheMu.RUnlock()
		return s, nil
	}
	cacheMu.RUnlock()

	var schemaDoc any
	if err := json.Unmarshal(schemaJSON, &schemaDoc); err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(key, schemaDoc); err != nil {
		return nil, err
	}
	s, err := c.Compile(key)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	schemaCache[key] = s
	cacheMu.Unlock()
	return s, nil
}

// 用 JSON Schema 校验 data（schemaJSON/data 均为 JSON 字节），不通过时返回校验错误。
func Validate(schemaJSON, data []byte) error {
	schema, err := getSchema(schemaJSON)
	if err != nil {
		return err
	}
	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		return err
	}
	return schema.Validate(instance)
}
