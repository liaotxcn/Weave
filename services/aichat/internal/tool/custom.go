/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tool

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// customTool 自定义工具示例
type customTool struct{}

// NewCustomTool 创建一个自定义工具示例
func NewCustomTool() tool.BaseTool {
	return &customTool{}
}

func (t *customTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "weave_tool",
		Desc: "这是一个自定义工具示例",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"param1": {
				Type: schema.String,
				Desc: "参数1",
			},
			"param2": {
				Type: schema.Integer,
				Desc: "参数2",
			},
		}),
	}, nil
}

func (t *customTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 工具调用的实现逻辑
	return `{"status":"ok","result":"自定义工具调用成功"}`, nil
}
