<p align="center">
  <img src="assets/logo.png" width="140" alt="noddock logo">
</p>

<h1 align="center">noddock</h1>

<p align="center">
  <a href="README.md">English</a> | <a href="README.zh-CN.md">中文</a>
</p>

<p align="center">
  <a href="https://noddock.com"><img src="https://img.shields.io/badge/官网-noddock.com-orange" alt="官网"></a>
  <a href="https://github.com/noddock/noddock/releases"><img src="https://img.shields.io/badge/下载-release-blue" alt="下载"></a>
  <a href="https://github.com/noddock/noddock/issues"><img src="https://img.shields.io/badge/反馈-bug%20%26%20feature-red" alt="Issues"></a>
  <a href="https://www.youtube.com/@noddock-news"><img src="https://img.shields.io/badge/youtube-@noddock--news-red" alt="YouTube"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-green" alt="License"></a>
</p>

## 简介

**noddock** 是一款节点图工作流自动化工具：通过连接节点搭建工作流，组合工作流为项目。提供本地、稳定、安全、透明的自动化服务、并且提供有自定义JS、Python节点和自定义节点UI的灵活拓展功能和一键导出和加载的分享功能。你需要携带自己的API来使用相关的模型节点。

本仓库是 noddock 的**开源部分**，包含：

- 全部内置**节点实现**（`nodes/`）——应用使用的完整节点目录和实现方法；
- 简单的 **mock 运行时**（`runtime/`）——单独编译并运行单个节点；
- 一个**示例程序**（`cmd/noddock`）——完整演示单节点运行：从注册表加载、设置参数、运行、打印结果。

生产引擎（项目/工作流管理、前端、账号与保险箱等系统）**不包含**在本仓库中。

## 下载

noddock 应用通过 GitHub Releases 分发：

👉 **[下载最新版本](https://github.com/noddock/noddock/releases)**

## 仓库结构

| 路径               | 说明                                                                                               |
| ------------------ | -------------------------------------------------------------------------------------------------- |
| `nodes/`           | 内置节点实现（LLM、MCP、exec、db、json、logic、stream、audio、image、ui、core）。                  |
| `runtime/`         | mock 运行时：节点 spec/参数/上下文/流类型、注册表、单节点运行器。                                  |
| `cmd/noddock/`     | 示例程序：运行单个节点（Json--InlineJS 演示）并打印结果上下文。                                    |
| `example_project/` | 安装时自带的示例项目，包含教程提及的参考工作流。                                                   |
| `assets/`          | 软件自带的素材，包括可用的JS脚本和官方的自定义HTML模板，noddock中可直接通过/assets/...的方式访问。 |

## 快速开始

```sh
# 运行内置演示（Json--InlineJS 节点）
go run ./cmd/noddock
```

`cmd/noddock` 是一个**示例程序**，展示选择一个节点并赋值，然后运行获取结果的全过程。更改main代码并重新编译以选取不同节点或测试新节点。
（见 `cmd/noddock/main.go`）：

1. 从注册表加载节点：`manager.Registry["Json--InlineJS"]`；
2. 构建 `graph.Context`——先用 spec 默认值填充
3. 用 `ctx.Set` 设置自己的输入/编辑值
4. 运行 `node.Run(ctx, &runTool{node})`，从 `ctx.Data` 读回结果。

`cmd/noddock/node_run_tool.go` 里是一个mock的 `NodeRunTool`，提供与节点本身无关的其他系统的方法。

## Bug 反馈与功能建议

- 如果希望提交bug请通过 [提交 issue](https://github.com/noddock/noddock/issues)或者发送邮件到bug-report@noddock.com
- 修改建议或者新功能提议请通过issue提交。
- 其他联系请参考noddock.com的contact页面。

---

> nooddock 应用基于 [Wails v3](https://wails.io) 构建——使用 Web 前端 + Go
> 后端开发桌面应用的框架。
