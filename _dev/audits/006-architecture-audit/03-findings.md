# Change 006 Architecture Findings

> Finding 按用户影响、修改风险与范围约束分类。

| ID | Finding | 严重等级 | 用户影响 | 预计修改范围 | 分类 | 推荐进入 Change 006 |
|---|---|---:|---|---|---|---|
| F-006-01 | SSE 初始快照读取与订阅注册不原子 | P1 | 手机在建连瞬间若恰逢 done/error 发布，可能永久显示旧状态，直到下一题产生事件 | `pkg/mobile/bridge.go`、`server.go`、测试 | `SUITABLE FOR CHANGE006` | **是** |
| F-006-02 | WindowVisible 各入口是否绕过状态机 | P3 | 未发现绕过；Tray、Shortcut、SecondInstance、auto-hide 均统一 | 无 | `ALREADY IMPLEMENTED` | 否 |
| F-006-03 | Windows `SetWindowLevel` 为空实现 | P3 | 当前前端仅在 macOS 截图权限流程调用，Windows 用户路径不可达 | 平台层 | `DOCUMENTATION ONLY` | 否 |
| F-006-04 | Shortcut `maps.Copy` 不删除旧 action | P3 | 当前 UI 无删除绑定能力，实际保存是覆盖；暂无可复现用户缺陷 | `pkg/shortcut` | `DEFER` | 否 |
| F-006-05 | Solver 字段无 mutex | P2 | 主解题路径串行，但运行中保存配置可能与 provider/history 访问竞态 | `pkg/solution`、`app`，需锁粒度设计 | `DEFER` | 否 |
| F-006-06 | KeepContext 保存 multipart base64 | P2 | 最多 10 轮仍可能占较多内存；这是已确认的保真语义 | Solver/history 策略 | `DEFER` | 否 |
| F-006-07 | Mobile 不显示截图采集阶段 | P3 | 手机从 Provider 开始时才进入 waiting；符合既有协议白名单和“不传截图”决策 | 协议/路由 | `DOCUMENTATION ONLY` | 否 |
| F-006-08 | ModelSelect 不过滤非 image 模型 | P2 | 用户可能选择文本模型导致请求失败；但强制过滤会隐藏未知/自定义视觉模型 | 前端产品策略与 capability 数据 | `DEFER` | 否 |
| F-006-09 | MobileInfo 与 hidden 模式终态反馈 | P3 | Port、URL、available 及 `solution-hidden` 成功/错误反馈均已接线 | 无 | `ALREADY IMPLEMENTED` | 否 |
| F-006-10 | OutputMode 三态 UI/保存/auto-hide | P3 | desktop/mobile/dual 可保存；进入 mobile 时服务可用才 one-shot Hide | 无 | `ALREADY IMPLEMENTED` | 否 |

## F-006-01 根因

当前 SSE 建连顺序为：

1. `bridge.GetState()` 读取快照 S(n)。
2. handler 将 S(n) 写给客户端并 Flush。
3. `bridge.Subscribe()` 注册 channel。

若 `Publish()` 在步骤 1 与步骤 3 之间生成 S(n+1)，该客户端既收到旧 S(n)，又没有订阅到 S(n+1)。若 S(n+1) 是本题终态且之后没有新事件，手机会一直停留在旧 waiting/streaming/idle。

## 推荐修复边界

- 在 `Bridge` 同一把锁内完成“创建订阅 channel、注册订阅、复制当前 Snapshot”。
- Server 必须先取得已注册订阅和初始 Snapshot，再 Flush 初始事件。
- 保持现有 SSE event 名、Snapshot JSON、buffer 大小、慢客户端丢帧策略、多客户端广播与鉴权不变。
- 增加测试，明确断言第一次 Flush 时订阅已注册，并保留现有端到端握手/发布测试。