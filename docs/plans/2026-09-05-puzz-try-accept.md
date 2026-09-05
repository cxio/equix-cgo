# puzz Try / Accept 实现 Plan

> **给 agentic workers：** 必需子技能：使用 subagent-driven-development（推荐）或 executing-plans 逐任务实现此 plan。步骤使用复选框（`- [ ]`）语法进行跟踪。

**Goal:** 在 `puzz.Threshold` 上新增无 Nonce 一轮原语 `Try` / `Accept`，难度仍用命中率，现有 nonce 搜索 API 不变。

**Architecture:** 门槛统一为 `SHA256(seed || solution)`。`Try` 调用 `equix.Solve(seed)` 取第一个过门槛的解；`Accept` 先哈希再 `equix.Verify`。将 `combinedHash` 改为与 `seedHash(challenge || le64(nonce), sol)` 共用同一条 SHA-256，但 `Solve*` / `Verify*` 控制流与主包入口不变，不改写成调用 `Try` / `Accept`。

**Tech Stack:** Go 1.27、cgo、模块 `github.com/cxio/equix-cgo`（puzz 只调用主包，不 `import "C"`）。

## 全局约束

- 模块路径：`github.com/cxio/equix-cgo`
- 注释、文档、commit message 用简体中文；错误消息用英文
- 不改现有函数签名，不删 `Nonce`，不改 24 字节编解码
- 不新增 `TryContext`、`TryWithHashes`、`AcceptWithHashes`
- 不引入第二种 `Solution` 类型，不在 puzz 内重导出 `equix.Solution`
- 不把 `Accept` 改成返回 `error`（继续用 `bool`）
- 不新增 `Solver` / `Verifier`；继续走主包池化入口
- 不修改 `third_party/`、`internal/native/` 或主包公开 API
- 不把现有 `Solve` / `SolveContext` 改写成调用 `Try`；`SolveWithHashes*` 仍走 `equix.SolveWithHashesAndNonce`
- `CGO_ENABLED=1`；验证命令：`go vet ./... && go test ./...`

## 文件结构

| 路径 | 职责 |
| --- | --- |
| `puzz/puzzle.go` | 现有阈值/求解/校验；本 plan 抽 `seedHash`，追加 `Try` / `Accept` |
| `puzz/puzzle_test.go` | 现有测试；本 plan 追加 Try/Accept 用例与测试辅助 |
| `puzz/doc.go` | 包注释，链到 `Try` / `Accept` |
| `puzz/README.md` | 用法，补无 Nonce 一节 |

不新建文件。不改根 `README.md`。

---

### Task 1: seedHash 与 Try

**Files:**
- Modify: `puzz/puzzle.go`（`combinedHash` 改为走 `seedHash`；在 `FromBits` 之后、`hit` 附近追加 `seedHash` / `appendLE64`；在 `Solve` 之前追加 `Try`）
- Test: `puzz/puzzle_test.go`（在 `TestSolveAndVerify` 之前追加辅助函数与 `TestTry*`）

**Interfaces:**
- Consumes: 现有 `Threshold`、`hit`、`equix.Solve`、`equix.Solution`
- Produces:
  - `func seedHash(dst *[32]byte, seed []byte, sol equix.Solution)`（未导出）
  - `func appendLE64(challenge []byte, nonce uint64) []byte`（未导出；测试与 `combinedHash` 共用）
  - `func (th Threshold) Try(seed []byte) (*equix.Solution, error)`

- [ ] **Step 1: 编写失败的测试**

在 `puzz/puzzle_test.go` 的 `TestSolveAndVerify` 之前插入：

```go
func findSolvedSeed(t *testing.T, prefix string) []byte {
	t.Helper()
	for i := 0; i < 256; i++ {
		seed := append([]byte(prefix), byte(i))
		sols, err := equix.Solve(seed)
		if err != nil {
			t.Fatal(err)
		}
		if len(sols) > 0 {
			return seed
		}
	}
	t.Fatal("no Equi-X solutions in 256 seeds")
	return nil
}

func TestTryHitMatchesSolve(t *testing.T) {
	seed := findSolvedSeed(t, "equix-cgo/puzzle try hit")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}

	sol, err := th.Try(seed)
	if err != nil {
		t.Fatalf("Try: %v", err)
	}
	if sol == nil {
		t.Fatal("Try returned nil under FromBits(0)")
	}

	want, err := equix.Solve(seed)
	if err != nil {
		t.Fatal(err)
	}
	if *sol != want[0] {
		t.Fatalf("Try = %v, want equix.Solve(seed)[0] = %v", *sol, want[0])
	}
}

func TestTryMiss(t *testing.T) {
	for i := 0; i < 32; i++ {
		seed := findSolvedSeed(t, fmt.Sprintf("equix-cgo/puzzle try miss %d", i))
		sol, err := Threshold(0).Try(seed)
		if err != nil {
			t.Fatalf("Try: %v", err)
		}
		if sol == nil {
			return
		}
	}
	t.Fatal("Try(Threshold(0)) kept hitting")
}

func TestTryEmptySeed(t *testing.T) {
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range [][]byte{nil, {}} {
		sol, err := th.Try(seed)
		if err != nil {
			t.Fatalf("Try(%#v): %v", seed, err)
		}
		if sol != nil {
			// 本任务尚未实现 Accept；有解只断言非空且 Equi-X 校验通过。
			if err := equix.Verify(seed, *sol); err != nil {
				t.Fatalf("Try(%#v) returned unverifiable sol: %v", seed, err)
			}
		}
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `CGO_ENABLED=1 go test ./puzz -run 'TestTryHitMatchesSolve|TestTryMiss|TestTryEmptySeed' -count=1`

Expected: 编译失败，提示 `Threshold.Try` 未定义。

- [ ] **Step 3: 编写最小实现**

在 `puzz/puzzle.go` 把 `combinedHash` 换成零额外堆分配的共享哈希（`sha256.New` 与现在相同）。在 `hit` 之前插入 `appendLE64` 与 `seedHash`，并把 `combinedHash` 改为调用它们：

```go
// appendLE64 返回 challenge 后接 8 字节 little-endian nonce，
// 与主包 Equi-X 的 nonce 拼接约定一致。
func appendLE64(challenge []byte, nonce uint64) []byte {
	out := make([]byte, len(challenge)+8)
	copy(out, challenge)
	binary.LittleEndian.PutUint64(out[len(challenge):], nonce)
	return out
}

func encodeSol(dst *[16]byte, sol equix.Solution) {
	for i, v := range sol {
		binary.LittleEndian.PutUint16(dst[2*i:], v)
	}
}

// seedHash 计算 SHA256(seed || solution) 并写入 dst。
// solution 为 16 字节 little-endian 索引，与 equix.Solution.MarshalBinary 相同。
func seedHash(dst *[32]byte, seed []byte, sol equix.Solution) {
	var tail [16]byte
	encodeSol(&tail, sol)
	h := sha256.New()
	h.Write(seed)
	h.Write(tail[:])
	h.Sum(dst[:0])
}

// combinedHash 计算 SHA256(challenge || nonce || solution) 并写入 dst。
// 等价于 seedHash(challenge || le64(nonce), sol)；后缀定长，无编码歧义。
func combinedHash(dst *[32]byte, challenge []byte, nonce uint64, sol equix.Solution) {
	var nonceLE [8]byte
	binary.LittleEndian.PutUint64(nonceLE[:], nonce)
	var tail [16]byte
	encodeSol(&tail, sol)
	h := sha256.New()
	h.Write(challenge)
	h.Write(nonceLE[:])
	h.Write(tail[:])
	h.Sum(dst[:0])
}
```

删除旧的 `combinedHash` 函数体（不要保留两份）。`appendLE64` 本任务可先写上，Task 2 的测试会用到；`combinedHash` **不要**调用 `appendLE64`（避免在 `Solve` 热路径上多一次堆分配）。

在 `UnmarshalBinary` 之后、`Solve` 之前插入：

```go
// Try 对 seed 做一轮 Equi-X 求解，按主包返回顺序取第一个过门槛的解。
// 未命中（无解或候选都未过门槛）返回 nil, nil；只有主包内部错误才返回 error。
// seed 原样交给 equix.Solve，不拼接 nonce。
func (th Threshold) Try(seed []byte) (*equix.Solution, error) {
	sols, err := equix.Solve(seed)
	if err != nil {
		return nil, err
	}
	var hash [32]byte
	for i := range sols {
		seedHash(&hash, seed, sols[i])
		if th.hit(&hash) {
			sol := sols[i]
			return &sol, nil
		}
	}
	return nil, nil
}
```

`Try` 只调用 `equix.Solve`，不得调用 `SolveWithHashes*`。

- [ ] **Step 4: 运行测试验证通过**

Run: `CGO_ENABLED=1 go test ./puzz -run 'TestTryHitMatchesSolve|TestTryMiss|TestTryEmptySeed|TestSolveAndVerify|TestThresholdHit' -count=1`

Expected: 通过。`TestSolveAndVerify` 确认 `combinedHash` 重构未破坏 nonce 路径。

- [ ] **Step 5: Commit**

```bash
git add puzz/puzzle.go puzz/puzzle_test.go
git commit -m "$(cat <<'EOF'
为 puzz 增加无 Nonce 一轮求解 Try。

EOF
)"
```

---

### Task 2: Accept

**Files:**
- Modify: `puzz/puzzle.go`（在 `Try` 之后追加 `Accept`）
- Test: `puzz/puzzle_test.go`（在 `TestTryEmptySeed` 之后追加 Accept 用例；并把 `TestTryHitMatchesSolve` 补上 `Accept` 为 true）

**Interfaces:**
- Consumes: Task 1 的 `seedHash`、`appendLE64`、`Try`；`equix.Verify`；现有 `Solve` / `Verify`
- Produces:
  - `func (th Threshold) Accept(seed []byte, sol equix.Solution) bool`

- [ ] **Step 1: 编写失败的测试**

将 `TestTryHitMatchesSolve` 在 `*sol != want[0]` 断言之后追加：

```go
	if !th.Accept(seed, *sol) {
		t.Fatal("Accept rejected a freshly tried solution")
	}
```

在 `TestTryEmptySeed` 之后插入：

```go
func TestAcceptRejects(t *testing.T) {
	seed := findSolvedSeed(t, "equix-cgo/puzzle accept reject")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}
	sol, err := th.Try(seed)
	if err != nil || sol == nil {
		t.Fatalf("Try: sol=%v err=%v", sol, err)
	}

	bad := *sol
	bad[0] ^= 1
	if th.Accept(seed, bad) {
		t.Fatal("Accept accepted a tampered solution")
	}
	if th.Accept([]byte("equix-cgo/puzzle accept reject!"), *sol) {
		t.Fatal("Accept accepted a mismatched seed")
	}

	var hash [32]byte
	seedHash(&hash, seed, *sol)
	if Threshold(0).Accept(seed, *sol) && binary.BigEndian.Uint64(hash[:8]) != 0 {
		t.Fatal("Accept(Threshold(0)) accepted a non-zero digest")
	}
}

func TestAcceptMatchesVerifyNonce(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle accept nonce")
	th, err := FromProbability(0.1)
	if err != nil {
		t.Fatal(err)
	}
	sol, err := Solve(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	seed := appendLE64(challenge, sol.Nonce)
	got := th.Accept(seed, sol.Solution)
	want := Verify(challenge, th, sol)
	if got != want || !got {
		t.Fatalf("Accept=%v Verify=%v, want both true", got, want)
	}
}
```

把 `TestTryEmptySeed` 里「有解则 `equix.Verify`」改成「有解则 `th.Accept(seed, *sol)`」。

- [ ] **Step 2: 运行测试验证失败**

Run: `CGO_ENABLED=1 go test ./puzz -run 'TestTryHitMatchesSolve|TestAcceptRejects|TestAcceptMatchesVerifyNonce' -count=1`

Expected: 编译失败，提示 `Threshold.Accept` 未定义。

- [ ] **Step 3: 编写最小实现**

在 `Try` 之后插入：

```go
// Accept 校验 seed 上的解：先做 SHA-256 门槛，通过后再做 Equi-X 结构校验。
// 门槛未过或 Equi-X 失败（含内部错误）返回 false。
func (th Threshold) Accept(seed []byte, sol equix.Solution) bool {
	var hash [32]byte
	seedHash(&hash, seed, sol)
	if !th.hit(&hash) {
		return false
	}
	return equix.Verify(seed, sol) == nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `CGO_ENABLED=1 go test ./puzz -count=1`

Expected: 全部通过（含现有 `*WithHashes`、取消、并发、`TestSolveCost`）。`TestSolveCost` 约 10s。

- [ ] **Step 5: Commit**

```bash
git add puzz/puzzle.go puzz/puzzle_test.go
git commit -m "$(cat <<'EOF'
为 puzz 增加无 Nonce 一轮校验 Accept。

EOF
)"
```

---

### Task 3: 文档

**Files:**
- Modify: `puzz/doc.go`
- Modify: `puzz/README.md`（在「用法」之后、「取消与限时」之前插入「无 Nonce：调用方改 Challenge」）

**Interfaces:**
- Consumes: Task 1–2 的 `Try` / `Accept` 签名与未命中语义
- Produces: 文档与包注释

- [ ] **Step 1: 更新 doc.go**

在 `[Verify]` 那段之后、`[SolveWithHashes]` 那段之前插入：

```
// 调用方自己改 Challenge、不使用 nonce 时，用 [Try] / [Accept]：对当前 seed
// 只做一轮 Equi-X。[Try] 未命中返回 nil, nil。门槛公式的一般形式是
// SHA256(seed || solution)；nonce 路径的 seed 为 challenge || little-endian(nonce)。
```

完整包注释替换为：

```go
// Package puzz 在 Equi-X 求解之上叠加一层可调难度的 SHA-256 门槛，
// 提供“客户端搜索、服务端验证”的 PoW 谜题封装。
//
// 难度由 [Threshold] 表达：对每个候选解计算
//
//	SHA256(seed || solution) 的前 8 字节（大端 uint64）
//
// 该值小于等于 Threshold 即命中。nonce 路径的 seed 为
// challenge || little-endian(nonce)。两种构造方式对应同一机制：
//
//   - [FromProbability]：按期望命中概率构造（如 0.1 表示约 10% 的候选解命中）；
//   - [FromBits]：按组合哈希前缀零位数构造（如 4 位等价于 6.25% 命中概率）。
//
// [Solve] 从指定 nonce 起步、以素数 0x26f5 步进反复求解，直到命中难度。
// 搜索没有天然终点，需要限时或取消时使用 [SolveContext]。
//
// [Try] / [Accept] 不对 seed 拼接 nonce：调用方通过改变 Challenge 搜索。
// [Try] 只做一轮 Equi-X，未命中返回 nil, nil。
//
// [Verify] 先做廉价的哈希阈值比对（约百纳秒），通过后再做完整的 Equi-X
// 结构校验（约 10µs 量级），适合服务端对不可信提交的快速过滤。
//
// 需要同时得到解对应的 8 个 HashWX 哈希时，使用 [SolveWithHashes]、
// [SolveContextWithHashes] 与 [VerifyWithHashes]。哈希类型为
// [github.com/cxio/equix-cgo.Hashes]，作为多返回值，不进入 Solution 编码。
// 无 Nonce 路径需要哈希时，在 [Try] 命中后调用
// [github.com/cxio/equix-cgo.VerifyWithHashes]。
//
// 命中期望需要约 1/p（或 2^bits）个 Equi-X 候选解；Equi-X 每个 nonce 平均
// 产出约 1.7 个候选解，因此折合约 0.6/p（或 2^bits/1.7）轮 nonce 尝试。
//
// 所有函数内部经主包的池化入口工作，可从多个 goroutine 并发调用。
// 使用本包需要 CGO_ENABLED=1 和 C 编译器（与主包一致）。
package puzz
```

- [ ] **Step 2: 更新 README 机制段与新节**

把「机制」里的公式改成一般形式，并保留 nonce 特例：

```markdown
对每个候选解计算：

```text
SHA256(seed || solution)[:8] -> 大端 uint64
```

该数值 `<= Threshold` 即命中。nonce 搜索路径的 seed 为 `challenge || little-endian(nonce)`，因此哈希输入等价于 `challenge || nonce || solution`。两种构造方式对应同一机制：
```

在「用法」代码块之后、「取消与限时」之前插入：

```markdown
## 无 Nonce：调用方改 Challenge

`Try` / `Accept` 把 seed 原样交给 Equi-X，不拼接、不递增 nonce。未命中时换 Challenge 再试：

```go
th, err := puzz.FromBits(puzz.DefaultBits)
if err != nil {
    panic(err)
}

sol, err := th.Try(challenge)
if err != nil {
    panic(err)
}
if sol == nil {
    // 本轮未命中：调用方改 challenge 再 Try
    return
}

if !th.Accept(challenge, *sol) {
    panic("unexpected reject")
}

// 需要 HashWX 时走主包；本包不提供 TryWithHashes
h, err := equix.VerifyWithHashes(challenge, *sol)
_ = h
```

该示例需 `import "github.com/cxio/equix-cgo"`。
```

- [ ] **Step 3: 运行 vet 与全量测试**

Run: `CGO_ENABLED=1 go vet ./... && CGO_ENABLED=1 go test ./...`

Expected: 通过。`./puzz` 含计时类测试，全量约 10s。

- [ ] **Step 4: Commit**

```bash
git add puzz/doc.go puzz/README.md
git commit -m "$(cat <<'EOF'
补充 puzz 无 Nonce 一轮 Try/Accept 的文档。

EOF
)"
```
