# hash 知识库（合并版五件套）

组件：11 个一次性哈希计算函数（MD5/SHA1/SHA2 家族/SHA3 家族/SHA512 变体），输入 []byte 输出小写 hex 字符串。无状态纯函数。
更新日期：2026-10-12（刷新 @ d6b2b9c 复核：b1f8695→d6b2b9c 零代码变化，T/D/H 条目全部维持）。证据等级：A=代码现状/可回查提交，C=推测。

## 流程

```text
【计算路径】(hash.go，全部同构)
CalcX(content) → new hasher → h.Write(content) → hex.EncodeToString(h.Sum(nil))
无流式/分块 API：内容需一次性载入内存；无文件便捷函数；不返回 error（哈希不失败）

【算法清单】
MD5；SHA1；SHA224/256；SHA384/512（含 SHA512_224/512_256 截断变体）；SHA3_224/256/384/512（x/crypto/sha3）
安全位置说明：MD5/SHA1 带 //nolint:gosec 与「broken, 不用于安全用途」注释；SHA3 来自 golang.org/x/crypto（非标准库）
```

## 决策

- D-001 MD5/SHA1 保留供遗留兼容而非删除：nolint 注释明写 "provided for legacy compatibility"，文档注释声明不可用于安全场景（2026-10-12，A：代码现状）。重新评估：校验场景迁移完成后可移除或移入独立子包。
- D-002 SHA3 依赖 golang.org/x/crypto/sha3，而模块已声明 go 1.24.0（标准库已有 crypto/sha3）（2026-10-12，A：go.mod + 代码现状）。成因 (C)：推测为包创建于 2022 年，先于标准库 SHA3 落地，后续无人迁移。
- D-003 hex 字符串而非 []byte 返回值：一步到位的可读输出（2026-10-12，A：代码现状；(C) 动机推测为脚本/展示用途优先）。

## 历史

- H-001 演进三步：5a1e036（2022-10-14）SHA1 起家 → 33daede（同日）Get* 统一改名 Calc* → 26b3515（2023-08-03）扩至 11 个方法（A：提交记录）。当前影响：Get* 旧名不存在，历史代码需对表改名。
- H-002 9560adc（2025-11）为 gosec 合规补 nolint 与安全注释（A）。当前影响：lint 基线包含这些豁免，删 MD5/SHA1 时记得同步清理。

## 待办

- T-001 迁移 SHA3 至标准库 crypto/sha3，摆脱 x/crypto 依赖（状态：候选未验证，2026-10-12，A：go.mod go 1.24.0 + x/crypto import 并存）。影响：少一个外部依赖；输出应逐字节一致（同 Keccak）。完成条件：迁移 + known-answer 对比。
- T-002 大输入一次性载入内存（状态：候选未验证，2026-10-12，A：代码现状）。影响：GB 级文件场景内存峰值。完成条件：若确有需求再评估流式变体。

## 验收案例

- S-001 已知答案向量：对 "hello" 断言各 Calc* 输出（如 CalcSHA256 = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824）；空输入断言 SHA256 = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855。验证：直接字符串比较。
- S-002 输出形态：全部函数输出为小写 hex、长度与算法位宽匹配（MD5=32、SHA3_256=64 字符…）。验证：len + 字符集检查。
- S-003 与标准库一致性：CalcSHA3_256 与 crypto/sha3（或 x/crypto 参考实现）对随机 N 组输入逐字节一致。验证：循环对比。
