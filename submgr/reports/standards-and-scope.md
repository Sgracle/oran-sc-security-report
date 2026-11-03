> 这是 2026-10-27 完成的历史复核，打包日期为 2026-11-03。其他实现部分仅为当时指定快照的静态审查；未完成后续 FlexRIC dev 候选的动态验证。规范原文仅链接来源，本仓库不转载完整规范。

# E2 订阅归属问题：规范、实现与影响范围复核

完成日期：2026-10-27（Asia/Shanghai）。规范和公开源码材料于 2026-10-26 获取/核对。本次进行规范原文与源码静态复核，没有新增真实协议实验；原函数 mock 动态证据仍来自此前实验。

## 1. 结论与证据强度

**已确认的问题应归类为 O-RAN SC Subscription Manager 的实现层订阅归属校验缺失。没有证据表明 E2AP 规范设计必然允许一个节点删除其他节点的订阅。**

规范将该过程限定为请求删除发送 E2 Node 内已有的订阅，较新文本还明确 Request ID 和 Function ID 的节点作用域。SC 的处理却用全局 Instance ID 取得对象，未验证来源身份与该对象所有者的一致性，再使用所选对象的 MEID 生成删除。接收端已具有来源信息，在不修改 E2AP 报文格式的情况下可以修正这个实现。

具体分层：

| 问题 | 结论 |
|---|---|
| SC 审计版本存在归属校验缺失吗？ | 是，源码确认；I-release 原函数 mock 已动态观察错误关联、删除输出和模拟清理 |
| 只存在于 I-release 吗？ | 否；公开 H/I/J/L/M/master 固定快照中的七个关键函数内容一致，静态存在同一缺失路径 |
| 是协议强制产生的漏洞吗？ | 未发现这种证据；规范限定本节点订阅，现有来源上下文与字段足以实现正确校验 |
| 能证明只有 SC 存在这类问题吗？ | 不能；其他实现对同一消息支持不齐，并在相邻过程中发现同类绑定缺失的静态线索 |
| 已证明每个镜像/部署都可利用吗？ | 没有；镜像源码一致性、真实消息到达和端到端影响仍需验证 |

“SC 已确认缺陷”与“所有 O-RAN 实现均存在漏洞”是两个不同结论；本报告只支持前者及下述限定版本范围。

## 2. 规范原文核对

### 2.1 材料、出处和版本限制

| 文本 | 出处/定位 | 用途 |
|---|---|---|
| ETSI TS 104 039 V4.0.0 (2024-10)，对应 O-RAN.WG3.E2AP-R003-v04.00 | [ETSI 官方 PDF](https://www.etsi.org/deliver/etsi_ts/104000_104099/104039/04.00.00_60/ts_104039v040000p.pdf)，本目录保存 PDF、逐页文本和第 17/62 页截图 | 本次公开、可直接核验的主要规范依据 |
| O-RAN.WG3.E2AP-v02.03 | 本地 FlexRIC fork 仓库所带 PDF；第 19/20 页、§4.4、§9.2.7 | 对照较早主版本，不能替代精确 v02.00 正文 |
| O-RAN.WG3.E2AP-R003-v03.01 | 同仓库所带 PDF；第 18/19 页与第 65/66 页 | 检查过程和标识作用域的后续表述 |
| O-RAN.WG3.E2AP-R003-v05.00 | 本地 `standards` 目录 DOCX；重新从 `word/document.xml` 提取 | §8.2.2A、§9.2.7、§9.2.8 与公开 v4 文本相互印证 |
| O-RAN.WG3.TS.E2AP-R005-v09.00 | 本地 `standards` 目录 DOCX；重新提取 | 检查所持后续版本相同条款 |

两份 DOCX 重新提取的文本与既有 `official-E2AP-v05.00.txt`、`official-E2AP-v09.00.txt` 的 SHA-256 完全相同。原始文档散列、元数据与提取文件名记录在 [spec-metadata.json](../evidence/spec-metadata.json)。本地副本的原下载 URL 没有记录，不将文件名中的“official”视为独立来源认证；公开 ETSI 原文提供了可核验的主要依据。

SC 的生成头文件注明源自 `spec/e2ap-v02.00.00.asn`。本次未取得精确 v02.00 正文，故不能声称逐字核验了该版本的全部规范要求。v02.03 已有本节点过程范围、原订阅 Request ID 复用及无关联标识忽略的表述；v03.01、公开 v04.00 和本地 v05/v09 对节点作用域的表达更明确。下面将文字事实与工程解释分开。

### 2.2 直接可确认的条款

主要依据为 ETSI 官方 PDF 的以下位置：

| 条款 | 页码 | 原文可确认的语义 | 对本问题的意义 |
|---|---|---|---|
| §8.2.2A.1 General | 17 | E2 Node 发起，目的是删除该 E2 Node 中此前为 RIC 建立的既有订阅 | 不是授予节点跨 E2 Node 删除权限的操作 |
| §8.2.2A.2 Successful Operation | 17 | 引用成功订阅中分配的 Request ID；结合 Request ID 和 Function ID 关联订阅；RIC 可向该 E2 Node 发起删除 | 应保持请求来源、订阅及删除目标属于同一节点上下文 |
| §8.2.2A.3 Abnormal Conditions | 17 | 对未知的 Request ID/Function ID 标识给出忽略要求 | 不能仅因编号在另一个节点的表项中存在，就视为本来源的有效订阅 |
| §9.2.7 RIC Request ID | 62 | Request ID 对给定 E2 Node 唯一，包含 Requestor ID 与 Instance ID | Instance ID 单字段不是可脱离节点和完整身份使用的授权凭据 |
| §9.2.8 RAN Function ID | 62 | Function ID 在给定 E2 Node 内唯一 | 不同节点可以使用同样的 Function ID 数值，不能靠 Function ID 替代 owner 校验 |

§8.2.2A.3 原文使用 Request ID 与 Function ID 的组合描述。本报告不把其连词擅自扩写为所有单字段错误组合应返回的精确错误码规则。跨节点拒绝的核心依据是过程明确限定本节点已有订阅，以及接收端对有效关联的责任。

v02.03 的 §4.4 还要求 Indication、普通 Delete 和 Delete Required 复用对应成功 Subscription 的 Request ID；§8.2.2A.1 同样限定节点自身既有订阅；§8.2.2A.3 要求对不关联既有订阅的标识忽略。该版本 §9.2.7 仅描述 Request ID 字段，没有后来明确的“给定 E2 Node 唯一”句子；不能把后续新增措辞说成所有旧版本逐字相同。

### 2.3 工程解释与规范设计归因

工程上，安全关联应理解为：可信接入上下文中的 E2 Node 身份 + 成功订阅的完整 Request ID，并校验对应的 Function ID。可以使用按节点分表、复合键，或全局索引命中后检查 owner。规范没有规定必须使用哪种 map，也没有要求比较某个名为 MEID 的字符串。

全局唯一编号的分配策略本身可以成立；错误在于把“全局能查到对象”当作“发送者有权操作对象”。节点 A 和 B 的 Instance ID 即使完全不碰撞，A 引用 B 的编号仍可能触发此缺陷。

没有在 Delete Required 的 payload 中再次携带 Global E2 Node ID，不足以证明协议缺少身份信息。E2 接收连接已归属某个节点，SC 的 E2Term 又将该身份传入 RMR MEID。接收端应保留并使用这份可信来源上下文；仅对 payload 字段解码正确不能完成归属校验。连接认证也不能替代对象归属校验：认证为 A 的节点仍无权删除 B 的订阅。

规范可以进一步强化恶意已接入 peer 的跨节点负向场景、显式 MUST 绑定要求及一致性测试；这是可提出的安全澄清或加固建议。现有证据不支持把本次具体错误归为“符合规范也无法避免的协议设计漏洞”，也不证明规范所有安全要求都已完备。

## 3. O-RAN SC 成因与版本范围

### 3.1 实现中的具体偏差

基线数据流：

```text
连接来源 A -> params.Meid = A
                    |
消息引用 Instance ID 102
                    |
全局 register[102] -> owner B 的 Subscription
                    |
未核对 params.Meid 与 subs.Meid
                    |
从 subs 重建 MEID=B、Requestor=123、Instance=102、Function=subs.Function
                    |
生成对 B 的删除，并进入 B 的清理流程
```

原 handler 没有检查来源，也没有把输入完整 Request ID 和 Function ID 与持有订阅匹配。随后 `SendSubscriptionDeleteReq` 从 `subs` 取 MEID 和 Function，并将线上 Requestor ID 填成 123。后续连接状态检查针对的是被选中的 B，不能替代最初 A 对 B 的归属校验。

公开固定源文件可直接核验：[master handler](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/control.go#L1686)、[全局查询函数](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/registry.go#L606)、[重建删除参数](https://github.com/o-ran-sc/ric-plt-submgr/blob/98ebf666e797ee02a7b2946742ebcc4259252b4c/pkg/control/control.go#L1572)。

这里的固定线上值 123 属于 SC 当前实现，**不是规范常量**。通用修复应校验实际成功订阅使用的线上身份，不能把针对该快照的 123 比较原样推广到其他实现。

### 3.2 官方公开分支固定快照

从 `o-ran-sc/ric-plt-submgr` 公开 API 读取 branch→commit，再按固定 commit 下载 control.go、registry.go、container-tag.yaml，未切换原研究仓库分支。27 份文件全部成功取得，URL 和散列保存在 [sc-fetch-manifest.json](../evidence/sc-fetch-manifest.json)。

| 分支快照 | Commit | 源码 container-tag 声明 | 本条处理入口与判断 |
|---|---|---|---|
| e-release | `d5c027e9d52c5a1f43304a71cdf84b4c1ccf8024` | 0.8.2 | 所取 control.go 无 Required handler/分派，不证明整个版本不存在其他归属问题 |
| f-release | `2a08dd799ad0de2ec84f7dd32ef5a1c3974e2575` | 0.9.3 | 同上 |
| g-release | `1c54904eabebcae865d597b8ed34633482963093` | 0.9.5 | 同上 |
| h-release | `ad93392cb6c1180d9f4828baea4f5427c3163d4e` | 0.9.6 | 静态存在同一缺失路径 |
| i-release | `956bb478233a4467393a4f3f67f4b4c04544a736` | 0.10.1 | 同一缺失路径；此前对原函数做过 mock 动态验证 |
| j-release | `ff822852aa2d4a81b5b6da18111f55a9b315603e` | 0.10.2 | 静态存在同一缺失路径 |
| l-release | `ff822852aa2d4a81b5b6da18111f55a9b315603e` | 0.10.2 | 与 j-release 为同一 commit，非独立版本样本 |
| m-release | `191c81a6c587cde692345b50da94e7cdae8c56d8` | 0.10.3 | 静态存在同一缺失路径 |
| master | `98ebf666e797ee02a7b2946742ebcc4259252b4c` | 0.10.4 | 静态存在同一缺失路径 |

在 H/I/J/L/M/master 六个快照中，以下七个函数按 LF 读取后的函数全文 SHA-256 分别相同：Required handler、GetSubscriptionFirstMatch、SendSubscriptionDeleteReq、SendSubscriptionDeleteReqToE2T、handleSubscriptionDelete、sendE2TSubscriptionDeleteRequest、rmrSendToE2T。handler 的散列为 `ed86f38546ca1e727a67e9c63fe88c358e7e4e149d432d25be34ba39f63a6a46`。详见 [sc-version-evidence.json](../evidence/sc-version-evidence.json)。

这是**取样的分支 HEAD 与源码标签声明**，不是全部 Git 历史或所有发布镜像的验证。不能据此写“0.9.6 到 0.10.4 每个可能构建都可利用”，也没有确定最初引入 commit、全部补丁分叉或今后修复状态。

### 3.3 部署影响条件与范围

要在部署中产生对应影响，至少需满足：该消息能够到达相同处理逻辑；来源节点可提交相关消息；目标订阅位于同一 Submgr 注册域；订阅仍存在；目标标识可被引用；目标连接、valid/PolicyUpdate、端点和事务状态允许相应删除或清理路径。真实协议可达性和完整输出链仍需单独实测。

该 handler 没有 KPM 专属条件，故代码层风险不只限于 KPM，其他走相同订阅生命周期的功能也应检查。具体测量/业务影响取决于实际订阅内容，不能把删除订阅直接等同于基站断网或用户面中断。

基于 SC 组件构建的集成发行环境可能继承缺陷，不代表另一个独立 RIC 实现。本地 `oran-sc-ric` compose 的 `.env` 声明 Submgr=0.10.1、E2Term=6.0.4，可列为待核实对象；标签与源码版本相近不足以证明镜像二进制完全一致。

跨不同 RIC/Submgr 注册域的影响、未知订阅枚举成功率、merged 多端点和同订阅并发效果均未通过本次静态对比证明。

## 4. 其他实现复核

这些对比为只读静态审查，没有运行其他实现的协议实验；因此“没有相同 handler”不能写成“通过了该过程的安全校验”。

| 实现及锁定快照 | 同一 Delete Required 的支持情况 | 可支持的判断 |
|---|---|---|
| ONOS E2T，`onosproject/onos-e2t@7cdc37cfe8ae4020c455b2d44713e3ce44173574`，本地 master clean | 有 builder/decoder 及测试，但未接入 RIC southbound dispatcher；普通 Delete 仅匹配成功/失败响应，Required initiating 不会进入它 | 未发现 SC 同消息触发链；不能声称已有 owner 校验或整个 ONOS 安全 |
| FlexRIC fork，来源 `openaicellular/flexric@1f04cc558ebc8da9de6a620762bc02f5db4ecb4a`，本地 master clean | ASN 层有类型，v2/v3 应用类型/codec/handler 未接入；未知过程进入 assert/无效枚举路径 | 不具有 SC 的正常 Required 删除路径；断言和越界候选另属错误处理问题，不能记为安全拒绝 |
| srsRAN Project，`srsran/srsRAN_Project@d2f4b70dda8e2c557d5b05a0ac5f92dbddda19bc`，detached clean | E2 Node 角色；有 ASN 类型，业务目录未见 Required 发送实现 | 不是另一个汇聚多节点的 RIC，不能用它证明 RIC 归属校验正确 |

### 4.1 ONOS 的连接作用域与支持边界

`pkg/protocols/e2ap/server_conn.go:25-37` 对每条连接新建各 procedure 实例；77-94 是真实接收 dispatcher，无 Required 分支。`ric_subscription_delete.go:110-128` 只匹配普通 Delete 的 successful/unsuccessful outcome。

普通订阅/删除仍保留节点上下文：北向 `pkg/northbound/e2/v1beta1/subscription.go:239-245` 将 E2NodeID 放入 SubscriptionID；controller 在 `pkg/controller/v1beta1/subscription/controller.go:521-586` 从订阅 Master relation 获取具体连接，再在该连接发送删除。响应 map 虽只用 RequestorID，但 map 属于每连接 procedure，而非所有节点共用的一张表。

这提供了“索引还需看所在上下文”的架构对照，不是对 Required 的正向安全测试。另在 `pkg/southbound/e2ap/server/data_conn.go:51-66` 发现 Indication 通过共享 stream manager 按 RequestorID 查找，未显式比较该连接 E2NodeID；属于未完成审计的相邻候选，不能扩展成 ONOS 已确认受同漏洞影响。

### 4.2 FlexRIC fork 的相邻线索

`src/lib/e2ap/v2_03/dec/e2ap_msg_dec_asn.c:2686-2690` 的未知过程路径显式失败断言；若 NDEBUG 取消断言，会返回 NONE 枚举，随后 `:2719-2720` 用它索引 decoder 数组。v3 存在对应结构。这里只记录源码路径，未测试具体构建的运行结果，未将其定性为已复现 DoS。

普通响应/Indication 的来源绑定也值得另案验证：`src/ric/near_ric.c:425-434` 将解码消息交给 handler，未把 SCTP 来源节点传入；`src/ric/msg_handler_ric.c:155-158` 从响应中的 ID 查全局 pending；`src/ric/iApp/map_ric_id.c:121-130` 的相等判断只用 Requestor ID，尽管对象本身存有节点 ID；`msg_handler_iapp.c:126-155` 按该 ID 查询并在 Delete Response 后删除映射。

据此可提出普通响应或上报的同类来源绑定缺失候选，但还需要 pending/active 状态与动态可达性验证。这不是 Delete Required 的同一条删除攻击链，也没有验证最新 EURECOM upstream 或所有 FlexRIC 分叉。

### 4.3 srsRAN 的角色与 Function ID 观察

`lib/e2/common/e2_entity.cpp:34-50` 为每个 E2 entity 创建本地 subscription manager。`e2_subscription_manager_impl.cpp:66-69` 按 `(RequestorID, InstanceID)` 保存订阅；`:89-103` 普通删除按该二元 key 查询，FunctionID 只是回显，未与已保存值比对；`:132-139` 按相同 key 停止任务并删表。

因此它不能作为另一个 RIC 对跨节点 Required 正确处理的证明，也不能称已校验完整 RequestID+FunctionID。FunctionID 一致性是另一个静态观察，未在本次动态验证。

## 5. 对此前结论的修正

1. 跨节点归属缺失的结论保留，并有了规范语义和更多 SC 源码版本的支持。
2. 此前“远端失败/超时后本地仍清理”确有源码及 mock 行为证据，但**不能仅凭这个现象认定又发现一个协议安全漏洞**。v02.03 §8.2.2A.2（PDF 第 20 页）允许 RIC 释放必要资源并发起删除，文字顺序并未要求所有本地释放都等待远端确认。它是否进一步引起 ID 复用、误路由、恢复错误或其他安全影响，需要结合具体状态机另做验证。
3. 较新文本省去该显式 early-release 列项，但这也不足以推断所有早期实现的清理策略必然违规。该一致性现象应保留为待分析的生命周期路径，不与跨节点授权缺陷混为一谈。
4. E2AP 原文中没有逐字规定“比较 params.Meid 与 subs.Meid”，也没有规定 Requestor ID=123。检查的安全目的由过程范围支持；具体 guard 必须服从实现实际保存和发送的身份。
5. 不能从有限的其他实现支持缺口推出“只有 SC 存在任何订阅安全问题”。本次证实的是 SC 的这一条路径；其他实现相邻过程的线索尚处于静态候选阶段。

## 6. 修复归因与后续验证边界

现有协议下可采用：在 trusted E2 Node 上下文内查成功订阅；或全局查找后核对 owner、实际线上完整 RequestID 与 FunctionID，然后才调度删除/清理。校验应基于规范化的节点身份，不仅绑定某一条 SCTP socket，因为同一节点可以有多个合法传输关联。

本地 24 行修复符合当前 SC 快照的固定线上 Requestor 策略，并通过原函数 mock 回归；这并不证明真实 codec/外部库和完整原生包无回归。本次不更改此前源码补丁，也不部署。

还需完成的验证：精确 v02.00 规范文本；受影响分支的完整构建与真实协议路径；镜像 commit 映射；跨节点异常报文实际到达；真实路由/SDL/测量状态；其他实现候选的独立测试；SC 引入和修复历史。

综合归因：**协议定义了节点内订阅语义；SC 的具体实现未在内部全局资源索引上落实这一归属边界。当前应按实现缺陷报告和修复，而不应将其写成 O-RAN E2AP 协议普遍、不可避免的设计漏洞。**

## 7. 证据索引

- [规范材料与散列](../evidence/spec-metadata.json)，ETSI 逐页提取（原审计归档材料；本仓库未转载），第 17 页（原审计归档材料；本仓库未转载）、第 62 页（原审计归档材料；本仓库未转载）。
- SC 分支元数据（原审计归档材料；本仓库未转载）、[固定源码 URL/散列](../evidence/sc-fetch-manifest.json)、[函数级版本对比](../evidence/sc-version-evidence.json)。
- 此前完整实验报告（原审计归档材料；本仓库未转载）、[此前逐用例结果](../evidence/SUMMARY.md)。
- 本目录 `review-manifest.json` 记录本次关键证据散列、其他仓库 commit 和审查状态，独立于此前的不可改写证据清单。
