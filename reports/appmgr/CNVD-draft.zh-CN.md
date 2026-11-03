# CNVD 漏洞报送草稿

编制日期：2026-10-15
状态：源码审计发现；漏洞属性、动态复现和厂商确认待补充。
本文用于提交审核，不代表已获 CNVD 收录或分配 CVE 编号。

## 1. 基本信息

- 建议标题：O-RAN SC ric-plt-appmgr xapp_onboarder 存在 xApp 制品来源真实性校验缺失问题
- 厂商/维护组织：O-RAN Software Community（O-RAN SC）
- 产品：ric-plt-appmgr
- 组件：xapp_onboarder
- 建议归类：通用软件产品问题；完整性及来源验证缺失
- 建议 CWE：CWE-353（Missing Support for Integrity Check），以审核认定为准
- 风险等级：待评估。未证明攻击者权限、受支持部署的信任边界和实际影响前，不给出确定等级。
- 报告人/联系方式：[由提交人填写]
- 首次发现日期：[按原始审计记录填写，不以报告编制日替代]
- 已分配 CNVD 编号：无
- 正式 CVE 编号：尚未确认分配
- MITRE 申请跟踪号：CAN-2026-2039411（仅为申请跟踪号）
- 厂商协调工单：https://lf-o-ran-sc.atlassian.net/browse/RIC-1078

## 2. 产品及验证基线

ric-plt-appmgr 是 O-RAN SC 的 RIC 平台应用管理项目。本文关注其中将 xApp 描述符转换为 Helm Chart 的 xapp_onboarder 代码。

本次源码证据固定在：
- 仓库：https://github.com/o-ran-sc/ric-plt-appmgr
- 已审计提交：bdc6f07284d33aaad6a04e30fd5821776d7a8e18
- 包版本：xapp_onboarder 1.0.0（该提交的 setup.py）
- 先前审计记录关联标签：0.5.10；提交时应附标签解析结果核对。
- 影响起始版本、当前受支持版本范围、修复版本：尚未确认。

本文不把该固定提交称为“当前最新版本”，不将结论扩展至所有 O-RAN 实现、所有商用 RIC 或 srsran/oran-sc-ric。

## 3. 问题描述（可粘贴至漏洞描述栏）

在上述源码基线中，xapp_onboarder 接收 xApp 描述符后，对输入执行 JSON Schema 校验，随后生成、打包并分发 Helm Chart。审计的处理路径未发现将描述符绑定至受信发布者身份的签名验证步骤；描述符允许提交者设置镜像 registry、name、tag，以及容器 command 和 args，生成的 Deployment 使用 registry/name:tag 形式引用镜像。

该流程本身未建立描述符、预期发布者和最终执行镜像内容之间的密码学信任绑定。当不可信主体能够提交或修改被系统接受的 onboarding 输入，或者可变镜像标签指向的内容被替换，并且下游没有强制的可信来源检查时，未经可信发布者确认的制品可能进入可部署 Chart。若运维人员或自动化流程进一步部署该 Chart，容器将按实际获授的权限和网络访问范围运行。

本报告请求维护方确认该组件在受支持架构中的安全职责，以及是否存在无法绕过的外部验签/准入控制。如果入口仅允许具有等价部署权限的可信管理员使用，或下游强制校验能阻止不可信镜像生效，现有源码现象可能属于设计取舍或纵深防御建议，尚不足以单独证明可收录漏洞。

## 4. 源码证据

以下文件均为上述固定提交；2026-10-15 已重新读取关键源文件。

1. api/onboard.py：onboard() 执行 Draft7Validator.check_schema() 和 validate()，之后调用 xApp()、package_chart()、distribute_chart()；成功路径返回 201 Created。
https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/api/onboard.py

2. helm_controller/xapp_schema.py：镜像的必需字段为 registry、name、tag；支持 command 和 args；未将发布者签名或镜像 digest 列为上述制品身份的必需条件。
https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/helm_controller/xapp_schema.py

3. templates/deployment.yaml：image 按 registry/name:tag 渲染，并将描述符提供的 command、args 写入 Deployment。
https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/resources/xapp-std/templates/deployment.yaml

4. helm_controller/xApp_builder.py：package_chart() 调用 helm lint、helm package；distribute_chart() 上传 Chart。上述路径未发现发布者验签及镜像 digest 的强制信任绑定。
https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/xapp_onboarder/helm_controller/xApp_builder.py

5. tests/test_api.py 与 tests/constants.py：官方测试向 /api/v1/onboard 提交未包含发布者签名的测试数据，并断言 201 Created。它证明测试预期，不是本报告新执行的真实服务复现结果；测试替身也不能证明生产部署中的最终影响。
https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/tests/test_api.py
https://github.com/o-ran-sc/ric-plt-appmgr/blob/bdc6f07284d33aaad6a04e30fd5821776d7a8e18/xapp_orchestrater/dev/xapp_onboarder/tests/constants.py

## 5. 利用条件与影响边界

应在动态验证后填写攻击者的具体能力，例如：能够提交描述符的低权限账号、能够修改被下载描述符的数据源，或能够修改既有镜像标签的主体。当前尚未确认哪一种条件在受支持部署中成立。

需验证的边界是“不可信主体提供的制品进入受信部署流程”，不是管理员本来就能选择镜像并部署容器的常规能力。没有签名本身不等于已证明越权或远程代码执行。

可能影响包括 xApp 获授范围内的数据暴露、服务操作或资源消耗；这些均需结合实际 RBAC、网络策略、服务认证和凭据证明。当前不宣称未授权互联网攻击、Kubernetes 管理员权限、宿主机失陷、实际无线业务中断或对所有运营商的影响。

可变标签存在替换可能，但镜像是否重新拉取、何时生效，取决于实际缓存及 imagePullPolicy 等条件，需保留实测记录。

## 6. 隔离环境复现步骤（待执行并补齐实际结果）

环境字段待填写：
- 操作系统、Python/Helm/ChartMuseum 版本；
- 仓库提交与启动命令；
- 本地测试服务地址；
- 是否部署 Kubernetes、版本及命名空间；
- API 认证方式、测试身份和权限；
- 是否启用镜像验签、仓库限制、准入控制及具体配置。

A. 未签名制品入库验证：
1. 在隔离本地环境启动固定提交的 xapp_onboarder 和本地 Chart 仓库。
2. 以 tests/constants.py 中 mock_json_body 为基线，保留必需字段和 controls schema。
3. 将名称、版本改为独立测试值；镜像指向自有本地测试仓库中的无害镜像。镜像仅输出唯一标记，不读取凭据或发送外部网络请求。
4. 不添加发布者签名或受信发布者证明，向 /api/v1/onboard 发送请求，保存原始请求、响应、服务日志。
5. 若返回成功，下载生成 Chart，解包并渲染 Deployment，核对镜像、command、args 是否来自测试输入。
6. 删除一个必需字段（如 image.tag）再提交，记录 Schema 校验是否拒绝，作为输入校验工作的对照。
7. 分开记录“Chart 生成/入库成功”和“容器实际执行”；前者不等于后者。

B. 标签内容绑定验证：
1. 使本地镜像 tag 初始指向内容 A，保存 digest A 后生成 Chart。
2. 仅在自有仓库把同一 tag 指向内容 B，保存 digest B。
3. 在明确记录的拉取策略和缓存条件下，在隔离命名空间部署；保存实际 imageID 和无害标记。
4. 确认下游是否拒绝 B，以及是否存在强制签名或 digest 绑定。
5. 若身份验证已由外部控制有效阻止该路径，应如实记录，不能将其写成绕过成功。

C. 信任边界证明：
记录攻击者初始权限，证明其不能通过正常授权直接部署同等制品，但能通过本问题导致受信流程接受或运行该制品。若无法证明，应将报告定位为安全设计风险，等待维护方判断。

实际结果：
[尚未附真实服务的端到端 HTTP 回包、Chart 和运行记录。执行后逐项填写。不得把上述预期步骤描述为已经复现成功。]

## 7. 修复建议

在明确产品信任模型后：
- 对需来自受信发布者的 xApp，验证受信签名身份，并用签名清单绑定名称、版本、描述符、关联 schema 和每个镜像的 digest。
- 镜像引用固定为 registry/name@sha256:digest；digest 绑定内容，可信签名绑定身份，两者不能互相替代。
- 若由外部组件验签，应使该检查在受支持部署中强制、不可绕过，并覆盖入库到部署的完整路径。
- 对未签名、身份不受信、内容不匹配等输入明确拒绝并记录审计信息。
- 对上述失败情形补充测试，并结合 API 权限、准入控制和最小权限部署验证效果。

## 8. 协调和披露记录

- 已建立项目协调工单 RIC-1078；维护方是否确认漏洞、是否修复，尚未核实。
- 根据历史提交回执，2026-10-13 已向 MITRE CNA-LR 提交申请 CAN-2026-2039411，选择暂不公开；尚未确认正式 CVE 分配。
- 提交 CNVD 时如实填写已报送其他平台及已有公开链接，不勾选“从未向其他平台报送”。
- 完整细节此前是否公开，应按 RIC-1078 的实际可见内容填写，不仅因申请未公开就声称从未披露。
- 请求 CNVD 在与维护方协调后处理公开时间。平台是否支持该选择，以实际规则为准。

## 9. 提交前附件清单

已有：本文、固定提交的源码链接、MITRE 申请跟踪号、项目工单链接。
待补：版本和启动配置、测试身份/权限、真实请求响应、生成 Chart 与渲染结果、必要的无害运行证据、日期记录、截图。
建议使用清晰文件名；对附件计算 SHA-256 便于核验。
不要将私人凭据、令牌、无关用户数据放入附件。

## 10. CNVD 提交说明

入口：https://www.cnvd.org.cn/
应使用 CNVD 主站漏洞报送入口。zc.cnvd.org.cn 是授权项目众测平台，官方帮助说明其仅接收项目授权范围内漏洞。
官方帮助：https://zc.cnvd.org.cn/news/helpList

2026-10-15 检索时主站抓取返回访问错误，未能核实登录后表单的实时字段、附件格式和大小上限。因此下面是填报建议，不是当前界面逐字操作说明：
1. 打开主站，注册/登录；按页面实际要求完成账号资料和认证。
2. 进入“漏洞提交”或含义相同入口；本问题建议按通用软件产品填写，实际选项以页面为准。
3. 将本文标题、产品、版本、描述、源码证据、复现步骤、修复建议映射至对应栏。
4. 风险等级按实测条件填写；不要把以前草稿中的 CVSS 8.0 当作平台确认的等级。
5. 上传按页面格式要求整理的材料；姓名、联系方式、发现日期等由本人准确填写。
6. 在备注写明 RIC-1078、CAN-2026-2039411 和暂不公开的协调诉求。
7. 提交后保存平台返回的受理/流水号和回执；提交成功不等于确认收录或发证。
8. 在账户报送记录中查看反馈，补充材料时沿用原记录。

本草稿尚未向 CNVD 发送；当前证据不足以保证收录、定级或颁发原创漏洞证明。

