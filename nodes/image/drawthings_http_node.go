package nodes

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool/fo"
)

func init() {
	param_prompt := graph.Parameter{
		ID:      "prompt",
		Kind:    graph.ParamString,
		Default: "A high quality illustration, detailed, good lighting",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_negative_prompt := graph.Parameter{
		ID:      "negative_prompt",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput}, // 可接上游（如 LLM 生成的负面提示词），不接时用编辑值
	}
	param_init_images := graph.Parameter{
		ID:      "init_images",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_port := graph.Parameter{
		ID:      "port",
		Kind:    graph.ParamString,
		Default: "http://127.0.0.1:7860",
	}
	param_model := graph.Parameter{
		ID:      "model",
		Kind:    graph.ParamString,
		Default: "z_image_turbo.ckpt",
	}
	param_sampler := graph.Parameter{
		ID:      "sampler",
		Kind:    graph.ParamString,
		Default: "Euler A Trailing",
	}
	param_loras := graph.Parameter{
		ID:      "loras",
		Kind:    graph.ParamJSON,
		Default: `[]`,
	}
	param_strength := graph.Parameter{
		ID:      "strength",
		Kind:    graph.ParamFloat,
		Default: 1,
	}
	param_width := graph.Parameter{
		ID:      "width",
		Kind:    graph.ParamInt,
		Default: 512,
	}
	param_height := graph.Parameter{
		ID:      "height",
		Kind:    graph.ParamInt,
		Default: 1024,
	}
	param_seed := graph.Parameter{
		ID:      "seed",
		Kind:    graph.ParamInt,
		Default: -1,
	}
	param_steps := graph.Parameter{
		ID:      "steps",
		Kind:    graph.ParamInt,
		Default: 7,
	}
	param_guidance_scale := graph.Parameter{
		ID:      "guidance_scale",
		Kind:    graph.ParamFloat,
		Default: 1,
	}
	param_img_name := graph.Parameter{
		ID:      "img_name",
		Kind:    graph.ParamString,
		Default: "new_img",
	}
	param_save_path := graph.Parameter{
		ID:      "save_path",
		Kind:    graph.ParamString,
		Default: ".tmp", // 相对路径按工程目录解析（与 Base64ToFile 一致）。
	}
	param_settings_extra := graph.Parameter{
		ID:      "settings_extra",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnHidden},
	}

	param_output := graph.Parameter{
		ID:      "output_path",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Image--DrawThingsHTTP",
		Name:        "DrawThings (HTTP)",
		Description: `Calls the local DrawThings HTTP API to generate an image. input: prompt + size/sampler; output: base64 image. \n due to the limitation of the DT\'s http server, the init_image accept only one image. \n Lora eg: [{"file": "xxx.ckpt", "mode": "all", "weight": 0.8}]`,
		Plist: []*graph.Parameter{
			&param_prompt,
			&param_negative_prompt,
			&param_init_images,
			&param_port,
			&param_model,
			&param_sampler,
			&param_loras,
			&param_strength,
			&param_width,
			&param_height,
			&param_seed,
			&param_steps,
			&param_guidance_scale,
			&param_img_name,
			&param_save_path,
			&param_settings_extra,
			&param_output,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			port := ctx.Get(param_port.ID).(string)
			model := ctx.Get(param_model.ID).(string)
			sampler := ctx.Get(param_sampler.ID).(string)
			loras := ctx.Get(param_loras.ID).(json.RawMessage)
			width := ctx.Get(param_width.ID).(int)
			height := ctx.Get(param_height.ID).(int)
			prompt := ctx.Get(param_prompt.ID).(string)
			negative_prompt := ctx.Get(param_negative_prompt.ID).(string)
			seed := ctx.Get(param_seed.ID).(int)
			steps := ctx.Get(param_steps.ID).(int)
			strength := ctx.Get(param_strength.ID).(float64)
			guidance_scale := ctx.Get(param_guidance_scale.ID).(float64)
			init_images := ctx.Get(param_init_images.ID).(string)

			savePath := ctx.Get(param_save_path.ID).(string)
			imgName := ctx.Get(param_img_name.ID).(string)

			var lorasMsA any
			if err := json.Unmarshal(loras, &lorasMsA); err != nil {
				return fmt.Errorf("drawthings: loras must be valid JSON: %w", err)
			}

			var url string
			url = fmt.Sprintf("%s/sdapi/v1/txt2img", port)

			reqBody := map[string]any{
				"model":           model,
				"sampler":         sampler,
				"loras":           lorasMsA,
				"width":           width,
				"height":          height,
				"prompt":          prompt,
				"negative_prompt": negative_prompt,
				"seed":            seed,
				"steps":           steps,
				"guidance_scale":  guidance_scale,
				"strength":        strength,
			}
			// settings_extra：隐藏参数，非空时合并进请求 body（同名覆盖、新增添加）。
			extraJson, _ := ctx.Get(param_settings_extra.ID).(json.RawMessage)
			if len(extraJson) > 2 {
				var extra map[string]any
				if err := json.Unmarshal(extraJson, &extra); err != nil {
					return fmt.Errorf("drawthings: settings_extra must be a JSON object: %w", err)
				}
				for k, v := range extra {
					reqBody[k] = v
				}
			}
			if init_images != "" {
				url = fmt.Sprintf("%s/sdapi/v1/img2img", port)
				initPath, err := fo.GetPath(init_images, nRT.GetPID(), true, true)
				if err != nil {
					return err
				}
				imgData, err := os.ReadFile(initPath)
				if err != nil {
					return err
				}
				reqBody["init_images"] = []string{base64.StdEncoding.EncodeToString(imgData)}
			}

			data, _ := json.Marshal(reqBody)

			req, err := http.NewRequestWithContext(nRT.GetRtCtx(), "POST", url, bytes.NewBuffer(data))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 0} // 生图耗时长，不设超时；可被 interrupt 取消。
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var result struct {
				Images []string `json:"images"`
				Detail string   `json:"detail,omitempty"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			if len(result.Images) == 0 {
				if result.Detail != "" {
					return errors.New(result.Detail)
				} else {
					return fmt.Errorf("drawthings: no image returned")
				}
			}

			// 解码 base64
			imgBytes, err := base64.StdEncoding.DecodeString(result.Images[0])
			if err != nil {
				return err
			}

			// save_path：统一规则解析（相对→工程目录，.. 不允许；绝对原样）。
			absSavePath, err := fo.GetPath(savePath, nRT.GetPID(), true, true)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(absSavePath, os.ModePerm); err != nil {
				return err
			}

			// 生成文件名（img_name 是 LLM 可写输入，清洗防路径穿越）。
			filename := fmt.Sprintf("%s.png", sanitizeImgName(imgName))
			fullPath := filepath.Join(absSavePath, filename)

			// 写入文件
			if err := os.WriteFile(fullPath, imgBytes, 0644); err != nil {
				return err
			}

			// 输出保持 save_path 的形态（相对配置 → 相对路径，不把本机绝对路径带进上下文）。
			outPath := fullPath
			if !filepath.IsAbs(savePath) {
				outPath = filepath.Join(savePath, filename)
			}
			ctx.Set(param_output.ID, outPath)

			return nil
		},
	})
}

// sanitizeImgName 清洗 LLM 可写的 img_name：路径分隔符/特殊字符替换为 _，去掉 ..，空值回退默认名。
func sanitizeImgName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\n', '\r', '\t':
			return '_'
		default:
			return r
		}
	}, name)
	name = strings.ReplaceAll(name, "..", "")
	if name == "" {
		return "new_img"
	}
	return name
}
