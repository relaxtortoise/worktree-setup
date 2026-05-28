# Testing Strategy & Coverage Improvement Design Spec

## Overview

建立完善的测试体系，将项目整体覆盖率从 17.6% 提升到 80%+，核心包（config/git/actions/engine/worktree）达到 90%+。

采用分层测试策略：纯单元测试（std testing）+ 集成测试（testcontainers-go）+ TUI 组件测试（bubbletea）+ CLI 命令测试（cobra）。

### 目标覆盖率

| 优先级 | 包 | 当前 | 目标 |
|---|---|---|---|
| P0 | internal/config | 69.1% | 90%+ |
| P1 | internal/git | 0% | 90%+ |
| P2 | internal/actions | 31.6% | 90%+ |
| P3 | internal/engine | 0% | 90%+ |
| P4 | internal/worktree | 26.4% | 90%+ |
| P5 | internal/hooks | 0% | 90%+ |
| P6 | internal/tui | 0% | 80%+ |
| P7 | cmd/cli | 0% | 80%+ |

### 依赖层次与实施顺序

```
config (P0) → git (P1) → actions (P2) → engine (P3)
                                              ↓
                              worktree (P4) → hooks (P5)
                                              ↓
                                        tui (P6) → cmd/cli (P7)
```

## testcontainers 集成测试策略

### 容器方案

每包通过 `TestMain` 启动一个共享容器，包内所有测试共用，每个测试用例在独立子目录中操作。

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    req := testcontainers.ContainerRequest{
        Image:      "alpine/git:latest",
        Cmd:        []string{"tail", "-f", "/dev/null"},
        WaitingFor: wait.ForLog(""), // startup immediately
    }
    container, _ = testcontainers.GenericContainer(ctx, req)
    code := m.Run()
    container.Terminate(ctx)
    os.Exit(code)
}

func TestXxx(t *testing.T) {
    repoDir := initGitRepo(t, "test-xxx") // init bare git repo in tmpdir
    // ...
}
```

### 容器用途

- `internal/git`：在容器内或通过 bind mount 执行真实 git 命令（worktree add/remove/list、for-each-ref 等）
- `internal/actions`：执行真实文件复制、软链接、shell 命令
- `internal/worktree`：通过 testcontainer git 仓库执行 Create/Remove 全流程
- `internal/hooks`：在容器内 git 仓库的 .git/hooks 目录写入和验证

### 隔离模型

- 每个测试函数使用 `t.TempDir()` 作为独立 git 仓库
- 通过 `git.Run()` 直接在本地执行（无需 exec 到容器），但依赖 testcontainer 提供的隔离 git 环境
- 包内测试串行运行（go test 默认行为），避免 git 操作冲突

## Phase 0：CLI 逻辑下沉（重构前置）

### 移动到 `internal/config/`

| 函数 | 当前位置 | 说明 |
|---|---|---|
| `printConfigValue` | cmd/cli/config_cmd.go | config 值打印 |
| `setConfigValue` | cmd/cli/config_cmd.go | config 值设置 |
| `printConfigFile` | cmd/cli/config_cmd.go | config 文件输出 |
| `writeConfigFile` | cmd/cli/config_cmd.go | config 文件写入 |

导出为：`PrintValue`, `SetValue`, `PrintFile`, `WriteFile`

### 移动到 `internal/git/`

| 函数 | 当前位置 | 说明 |
|---|---|---|
| `urlToProjectName` | cmd/cli/root.go | URL→项目名解析 |
| `projectName` | cmd/cli/root.go | 获取远程仓库项目名 |

导出为：`URLToProjectName`, `ProjectName`

### 移动到 `internal/engine/`

| 函数 | 当前位置 | 说明 |
|---|---|---|
| `isNewWorktree` | cmd/cli/run.go | 判断新 worktree 创建 |

导出为：`IsNewWorktree`

### 移动到 `internal/selfupdate/`（新建）

| 类型/函数 | 当前位置 | 说明 |
|---|---|---|
| `githubRelease` | cmd/cli/selfupdate.go | GitHub release 结构 |
| `httpGet` | cmd/cli/selfupdate.go | HTTP GET 请求 |
| `getLatestRelease` | cmd/cli/selfupdate.go | 获取最新 release |
| `getReleaseByTag` | cmd/cli/selfupdate.go | 按 tag 获取 release |
| `runSelfUpdate` | cmd/cli/selfupdate.go | 自更新主逻辑 |
| `SelfUpdater` struct | 新建 | 封装自更新依赖（可注入 http.Client） |

### CLI 层保留内容

每个命令文件仅保留：
- `cobra.Command` 定义和 `Use`/`Short`/`Long`
- Flag 变量声明和 `init()` 中的 `Flags().BoolVar()` 绑定
- `RunE` 直接委托给 internal 包（3-5行）

## Phase 1：internal/config 69% → 90%

### 已有测试覆盖

- `ParseFile`：map 形式 YAML、steps 隐式 run、list 形式 copy
- `Merge`：高优先级覆盖、nil 跳过
- `LoadHierarchy`：三层配置合并

### 补充测试（~18个用例）

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestParseFile_FileNotFound` | `ParseFile` | 文件不存在返回 error |
| `TestParseFile_InvalidYAML` | `ParseFile` | 破损 YAML 返回 error |
| `TestParse_ValidYAML` | `Parse` | 直接解析 YAML 字节 |
| `TestParse_InvalidYAML` | `Parse` | 非法 YAML 返回 error |
| `TestParse_EmptyInput` | `Parse` | 空输入返回零值 Config |
| `TestPathStrategy_StringForm` | `PathStrategy.UnmarshalYAML` | 字符串形式 "sibling" |
| `TestPathStrategy_TemplateForm` | `PathStrategy.UnmarshalYAML` | 对象形式 {template: "..."} |
| `TestStep_ImplicitRun` | `Step.UnmarshalYAML` | 裸字符串 → 隐式 run |
| `TestStep_ObjectForm` | `Step.UnmarshalYAML` | 完整对象 {run:, copy:, symlink:} |
| `TestCopyItems_MapForm` | `CopyItems.UnmarshalYAML` | map 形式 copy |
| `TestCopyItems_EmptyList` | `CopyItems.UnmarshalYAML` | 空 list |
| `TestParseColonShorthand_Single` | `parseColonShorthand` | 单段（无冒号） |
| `TestParseColonShorthand_WithColon` | `parseColonShorthand` | 有冒号分隔 |
| `TestEvent_StepsOrLegacy_Steps` | `Event.StepsOrLegacy` | steps 字段存在时 |
| `TestEvent_StepsOrLegacy_Legacy` | `Event.StepsOrLegacy` | 三段式回退 |
| `TestMerge_PartialNilEvents` | `Merge` | 部分 Events 为 nil |
| `TestMerge_AllNil` | `Merge` | 全部参数为 nil |
| `TestLoadHierarchy_NoConfigExists` | `LoadHierarchy` | 配置文件不存在不报错 |
| `TestLoadHierarchy_BrokenYAML` | `LoadHierarchy` | 破损 YAML 被静默忽略 |
| `TestUserConfigPaths` | `UserConfigDir` 等 | 路径拼接正确性 |

### 不需要测试的部分

- `yaml.Unmarshal` 本身（第三方库行为）
- `os.ReadFile` / `os.MkdirAll` 等标准库

## Phase 2：internal/git 0% → 90%

### 可测试性改造

- `Run` 和 `RunInternal` 需要对 exec.Command 的依赖可注入
- 引入 `GitRunner` interface 或 `execCommand` 函数变量（与 CLI 层已有的 `execCommand` 模式一致）

### 测试方案

需要 testcontainer 提供真实的 git 环境：

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestRun_Success` | `Run` | 正常执行 git 命令 |
| `TestRun_Failure` | `Run` | 命令失败时的错误信息 |
| `TestRunInternal_WTInternal` | `RunInternal` | 验证 WT_INTERNAL 环境变量传递 |
| `TestListRemoteBranches` | `ListRemoteBranches` | 从 testcontainer git 仓库获取分支列表 |
| `TestListRemoteBranches_EmptyRepo` | `ListRemoteBranches` | 空仓库无分支 |
| `TestListWorktrees` | `ListWorktrees` | 解析 worktree list 输出 |
| `TestListWorktrees_WithBare` | `ListWorktrees` | 包含 bare worktree |
| `TestParsePorcelain` | `parsePorcelain` | porcelain 格式解析 |
| `TestAddWorktree` | `AddWorktree` | testcontainer 中添加 worktree |
| `TestRemoveWorktree` | `RemoveWorktree` | testcontainer 中删除 worktree |
| `TestFindMainWorktree_Main` | `FindMainWorktree` | 找到 main 分支 |
| `TestFindMainWorktree_Master` | `FindMainWorktree` | 找到 master 分支 |
| `TestFindMainWorktree_FirstNonBare` | `FindMainWorktree` | 无 main/master 时回退 |
| `TestCurrentWorktreePath` | `CurrentWorktreePath` | 返回当前路径 |
| `TestFetchOrigin` | `FetchOrigin` | testcontainer 中 fetch |
| `TestCheckedOutBranches` | `checkedOutBranches` | 解析已检出分支 |

## Phase 3：internal/actions 31.6% → 90%

### 已有测试

- `TestRunCommand`：单命令执行
- `TestCopyFiles`：单文件复制

### 补充测试

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestExecuteRun_DryRun` | `ExecuteRun` | dryRun=true 不执行 |
| `TestExecuteRun_CommandFailure` | `ExecuteRun` | 命令失败返回 error |
| `TestExecuteRun_EmptyCommands` | `ExecuteRun` | 空命令列表 |
| `TestExecuteCopy_Directory` | `ExecuteCopy` | 复制目录 |
| `TestExecuteCopy_SourceNotFound` | `ExecuteCopy` | 源文件不存在 |
| `TestExecuteCopy_MultipleItems` | `ExecuteCopy` | 多文件复制 |
| `TestExecuteSymlink_Success` | `ExecuteSymlink` | 创建软链接 |
| `TestExecuteSymlink_Glob` | `ExecuteSymlink` | 通配符（如果支持） |
| `TestRunner_ExecuteEvent_NilEvent` | `Runner.ExecuteEvent` | nil event |
| `TestRunner_ExecuteEvent_Steps` | `Runner.ExecuteEvent` | 多步骤执行 |
| `TestRunner_ExecuteEvent_StepFailure` | `Runner.ExecuteEvent` | 步骤失败错误传播 |
| `TestRunner_ExecutePreCreate` | `Runner.ExecutePreCreate` | pre-create 执行 |
| `TestRunner_ExecutePreCreate_NilEvent` | `Runner.ExecutePreCreate` | nil pre-create |
| `TestNewRunner` | `NewRunner` | 构造函数 |

## Phase 4：internal/engine 0% → 90%

Engine 是 Runner 的薄封装，测试重点在事件分发正确性：

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestNew` | `New` | 构造函数 |
| `TestRunPreCreate` | `RunPreCreate` | 有 pre-create 配置 |
| `TestRunPreCreate_NilOn` | `RunPreCreate` | On 为 nil |
| `TestRunPostCreate` | `RunPostCreate` | post-create 执行 |
| `TestRunPostCreate_NilOn` | `RunPostCreate` | On 为 nil |
| `TestRunPostCheckout` | `RunPostCheckout` | post-checkout 执行 |
| `TestRunPreDelete` | `RunPreDelete` | pre-delete 执行 |
| `TestRunPostDelete` | `RunPostDelete` | post-delete 使用 mainWorktree 路径 |
| `TestIsNewWorktree_AllZeros` | `IsNewWorktree` | 40个0 |
| `TestIsNewWorktree_ShortZeros` | `IsNewWorktree` | "0000..." 短格式 |
| `TestIsNewWorktree_RealCommit` | `IsNewWorktree` | 真实 commit hash |

## Phase 5：internal/worktree 26.4% → 90%

### 已有测试

- `ComputePath`：sibling/nested/custom template/nil strategy

### 补充测试

需要 testcontainer 提供隔离 git 仓库：

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestCreate_Success` | `Create` | 完整创建流程 |
| `TestCreate_ExplicitPath` | `Create` | 指定路径 |
| `TestCreate_AutoFetch` | `Create` | 自动 fetch |
| `TestCreate_NoMainWorktree` | `Create` | 自动检测 main worktree |
| `TestCreate_PreCreateHook` | `Create` | pre-create hook 执行 |
| `TestCreate_PostCreateHook` | `Create` | post-create hook 执行 |
| `TestRemove_Success` | `Remove` | 完整删除流程 |
| `TestRemove_Force` | `Remove` | force 删除 |
| `TestRemove_NoMainWorktree` | `Remove` | 自动检测 main worktree |
| `TestComputePath_Nested` | `ComputePath` | nested 策略 |
| `TestComputePath_Home` | `ComputePath` | home 策略（~ 展开） |
| `TestSanitizeBranch` | `sanitizeBranch` | 分支名 sanitize |

## Phase 6：internal/hooks 0% → 90%

使用 testcontainer 的隔离 git 仓库：

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestInstall` | `Install` | 安装 post-checkout hook |
| `TestInstall_DirNotExist` | `Install` | hooks 目录不存在 |
| `TestIsInstalled_True` | `IsInstalled` | 已安装 |
| `TestIsInstalled_False` | `IsInstalled` | 未安装 |
| `TestIsInstalled_CorruptedHook` | `IsInstalled` | hook 文件存在但无标记 |

## Phase 7：internal/tui 0% → 80%

TUI 测试使用 bubbletea 的内存测试模式：

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestModel_Init` | `model.Init` | 初始化命令 |
| `TestModel_Filter` | `model.Update` | 文本过滤 |
| `TestModel_CursorUp` | `model.Update` | 向上移动光标 |
| `TestModel_CursorDown` | `model.Update` | 向下移动光标 |
| `TestModel_Select` | `model.Update` | enter 确认 |
| `TestModel_Quit` | `model.Update` | esc 退出 |
| `TestModel_View_Empty` | `model.View` | 空列表视图 |
| `TestModel_View_Items` | `model.View` | 有项目时的视图 |
| `TestFormatTime_JustNow` | `formatTime` | 刚刚 |
| `TestFormatTime_Minutes` | `formatTime` | 分钟 |
| `TestFormatTime_Hours` | `formatTime` | 小时 |
| `TestFormatTime_Days` | `formatTime` | 天 |
| `TestFormatTime_Weeks` | `formatTime` | 周 |
| `TestFormatTime_Zero` | `formatTime` | 零值时间 |

## Phase 8：cmd/cli 0% → 80%

CLI 测试使用 cobra 命令测试模式 — 调用 `cmd.SetArgs()` + `cmd.Execute()` 验证输出和退出码：

| 测试函数 | 覆盖对象 | 场景 |
|---|---|---|
| `TestVersionCmd` | `versionCmd` | 输出版本信息 |
| `TestListCmd_NoWorktrees` | `listCmd` | 无 worktree |
| `TestAddCmd_MissingBranch` | `addCmd` | 缺少参数 |
| `TestRemoveCmd_MissingArg` | `removeCmd` | 缺少参数 |
| `TestInitCmd_NoGitRepo` | `initCmd` | 不在 git 仓库内 |
| `TestConfigCmd_Get_InvalidKey` | `configCmd` | 无效 key |
| `TestConfigCmd_Set_MissingValue` | `configCmd` | 缺少 value |
| `TestRunCmd_UnknownEvent` | `runCmd` | 未知事件类型 |

### 测试风格

统一使用 **table-driven cases**（表驱动测试），所有补充测试用例按照以下模式组织：

```go
func TestXxx(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"case 1", "input1", "expected1", false},
        {"case 2", "input2", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := DoSomething(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

每个 `t.Run` 子测试应可独立运行，不依赖全局状态。子测试名称应清晰描述场景。

## 工具与依赖

### go.mod 新增

```
require (
    github.com/testcontainers/testcontainers-go v0.x
    github.com/stretchr/testify v1.10.x
)
```

### 测试文件结构

```
internal/
  config/
    parser_test.go        # 已存在，补充用例
    hierarchy_test.go     # 已存在，补充用例
    schema_test.go        # 新建：UnmarshalYAML 专项测试
    paths_test.go         # 新建：路径函数测试
  git/
    git_test.go           # 新建：所有 git 操作测试
    main_test.go          # 新建：TestMain 容器启动
  actions/
    runner_test.go        # 已存在，补充用例
    copy_test.go          # 新建：ExecuteCopy 专项
    symlink_test.go       # 新建：ExecuteSymlink 专项
    run_test.go           # 新建：ExecuteRun 专项
  engine/
    engine_test.go        # 新建
    newworktree_test.go   # 新建：IsNewWorktree
  worktree/
    path_test.go          # 已存在，补充用例
    create_test.go        # 新建
    remove_test.go        # 新建
    main_test.go          # 新建：TestMain 容器启动
  hooks/
    installer_test.go     # 新建
    main_test.go          # 新建：TestMain 容器启动
  tui/
    selector_test.go      # 新建
  selfupdate/             # 新建包
    selfupdate.go         # 从 cmd/cli 下沉
    selfupdate_test.go    # 新建
cmd/cli/
  *_test.go               # 新建：各命令测试
```

## testcontainers 容器细节

### 镜像选择

使用 `alpine/git:latest`（~15MB），包含 git 和 busybox shell。不需要完整的 Linux 发行版。

### 工作模式

git 命令直接在宿主机执行（因为宿主机已有 git），但操作的文件系统通过 bind mount 连接到容器，容器确保隔离性。

```go
func setupGitRepo(t *testing.T, name string) string {
    t.Helper()
    dir := t.TempDir()

    // 初始化 git 仓库
    cmds := [][]string{
        {"git", "init", dir},
        {"git", "-C", dir, "config", "user.email", "test@test"},
        {"git", "-C", dir, "config", "user.name", "test"},
        {"git", "-C", dir, "commit", "--allow-empty", "-m", "initial"},
    }
    for _, args := range cmds {
        cmd := exec.Command(args[0], args[1:]...)
        if out, err := cmd.CombinedOutput(); err != nil {
            t.Fatalf("git setup %v: %s", args, out)
        }
    }
    return dir
}
```

对于需要 remote 的场景（`ListRemoteBranches`, `FetchOrigin`），在 testcontainer 中创建 bare repo 作为 remote，然后从本地仓库添加 origin 指向它。

## 实施顺序

1. **Phase 0**：CLI 逻辑下沉（确保不破坏现有功能）
2. **Phase 1**：config 测试补充（纯单元，无外部依赖）
3. **Phase 2~7**：按依赖层次推进，每完成一个包运行全量测试确保不引入回归
4. 每个 Phase 完成后运行 `go test ./... -cover` 验证覆盖率提升

## 非目标

- E2E 测试（不启动完整 CLI 进程）
- 性能/压力测试
- 跨平台测试（仅 Linux，CI 中 git 命令行为一致）
- 100% 覆盖率（对 TUI 和 CLI 层不强求）
