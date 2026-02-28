# directr flow

`flow` 包是在 `directr` 之上新增的一层流程引擎：

- 不依赖、也不修改现有 `cmd/directr-cli`。
- 通过 DSL 脚本定义可执行流程。
- 内置场景识别（required/forbidden/optional + score）。
- 提供 Python 友好的请求/响应桥接。

## 1. Go 内部调用

```go
definition, err := flow.LoadDefinitionDSL("pkg/flow/example/login_flow.flow")
if err != nil {
    return err
}

runner := flow.Runner{}
result, err := runner.Run(ctx, hwnd, definition)
if err != nil {
    // result.Steps 包含已执行步骤和最后状态
    return err
}
```

## 2. Flow Script DSL

一个最小可用示例见：`pkg/flow/example/login_flow.flow`。

核心概念：

- `flow <name> { ... }`: 定义整条流程。
- `state <name> { ... }`: 定义状态与对应动作。
- `scene { ... }`: 每个 state 下用于场景识别。
- `required/forbidden/optional`: 带有 selector 条件的场景规则。
- `transition to <State> { ... }`: 状态迁移。
- `click/fill/...`: 动作定义。

示例：

```text
flow demo-login-flow {
  initial: Login
  target: Dashboard
  maxSteps: 20

  state Login {
    scene {
      required automationId=username
      required automationId=password
      required name=登录 controlType=button
      forbidden name=退出登录
      optional name=menu-home weight=2
    }

    transition to Dashboard {
      click name=登录 controlType=button
      wait 500
    }
  }

  state Dashboard {
    scene {
      required name=工作台
    }
  }
}
```

## 3. Python 接入（推荐子进程）

新增独立入口：`cmd/directr-flow`（不影响现有 CLI）。

执行方式：

```powershell
directr-flow --request pkg/flow/example/run_request.json --pretty
```

或通过 stdin 传入：

```powershell
Get-Content pkg/flow/example/run_request.json | directr-flow --pretty
```

响应 JSON 包含：

- `result`: 执行结果与步骤。
- `error`: 失败时错误信息。

请求示例（脚本文本）：

```json
{
  "windowClass": "YourAppMainWindow",
  "windowTitle": "Your Application",
  "timeoutMs": 30000,
  "pollIntervalMs": 250,
  "snapshotDepth": 6,
  "snapshotNodes": 400,
  "script": "flow demo-login-flow { ... }"
}
```

请求示例（脚本文件）：

```json
{
  "windowTitle": "Your Application",
  "scriptPath": "pkg/flow/example/login_flow.flow"
}
```
