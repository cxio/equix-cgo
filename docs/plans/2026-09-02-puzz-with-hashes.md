# puzz 返回 HashWX Hashes 实现 Plan

> **给 agentic workers：** 必需子技能：使用 subagent-driven-development（推荐）或 executing-plans 逐任务实现此 plan。步骤使用复选框（`- [ ]`）语法进行跟踪。

**Goal:** 在 `puzz` 子包新增 `SolveWithHashes` / `SolveContextWithHashes` / `VerifyWithHashes`，把命中解对应的 8 个 HashWX 作为多返回值交出。

**Architecture:** 不改 `puzz.Solution`、不引入 `Result`。求解循环与现有 `SolveContext` 相同，仅将主包入口换成 `equix.SolveWithHashesAndNonce`。校验仍先做 SHA-256 门槛，通过后再走 `equix.VerifyWithHashesAndNonce`。现有 `Solve` / `SolveContext` 继续使用 `SolveWithNonce`，不得改走带哈希路径。

**Tech Stack:** Go 1.27、cgo、模块 `github.com/cxio/equix-cgo`（puzz 只调用主包，不 `import "C"`）。

## 全局约束

- 模块路径：`github.com/cxio/equix-cgo`
- 注释、文档、commit message 用简体中文；错误消息用英文
- 不给 `puzz.Solution` 增加 `Hashes` 字段，不改 24 字节编解码
- 不引入 `puzz.Result`，不在 puzz 内重导出 `equix.Hashes`
- 不把 `Verify` / `VerifyWithHashes` 改成返回 `error`（继续用 `bool`）
- 不在 puzz 内包装 `equix.VerifyHashes`
- 不新增 `Solver` / `Verifier` 类型；继续走主包池化入口
- 不修改 `third_party/`、`internal/native/` 或主包公开 API
- 现有 `Solve` / `SolveContext` **不得**改走 `SolveWithHashesAndNonce`
- `CGO_ENABLED=1`；验证命令：`go vet ./... && go test ./...`

## 文件结构

| 路径 | 职责 |
|------|------|
| `puzz/puzzle.go` | 现有阈值/求解/校验；本 plan 在此追加三个导出函数 |
| `puzz/puzzle_test.go` | 现有测试；本 plan 追加带哈希用例 |
| `puzz/doc.go` | 包注释，链到新函数 |
| `puzz/README.md` | 用法，补带哈希示例 |

不新建文件。

---

### Task 1: SolveWithHashes / SolveContextWithHashes

**Files:**
- Modify: `puzz/puzzle.go`（在 `SolveContext` 之后追加两个函数）
- Test: `puzz/puzzle_test.go`（在 `TestSolveAndVerify` 之后追加测试）

**Interfaces:**
- Consumes: 现有 `Threshold`、`combinedHash`、`hit`、`nonceStep`、`equix.SolveWithHashesAndNonce`、`equix.Hashes`
- Produces:
  - `func SolveWithHashes(challenge []byte, th Threshold, nonce uint64) (*Solution, equix.Hashes, error)`
  - `func SolveContextWithHashes(ctx context.Context, challenge []byte, th Threshold, nonce uint64) (*Solution, equix.Hashes, error)`

- [ ] **Step 1: 编写失败的测试**

在 `puzz/puzzle_test.go` 的 `TestSolveAndVerify` 之后插入：

```go
func TestSolveWithHashes(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle hashes")
	th, err := FromProbability(0.1)
	if err != nil {
		t.Fatal(err)
	}

	sol, h, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}
	if sol == nil {
		t.Fatal("SolveWithHashes returned nil solution")
	}
	if !Verify(challenge, th, sol) {
		t.Fatal("Verify rejected a freshly solved puzzle")
	}
	if err := equix.VerifyHashes(h); err != nil {
		t.Fatalf("VerifyHashes: %v", err)
	}
	want, err := equix.VerifyWithHashesAndNonce(challenge, sol.Nonce, sol.Solution)
	if err != nil {
		t.Fatalf("VerifyWithHashesAndNonce: %v", err)
	}
	if h != want {
		t.Fatalf("hashes mismatch: got %v want %v", h, want)
	}
}

func TestSolveWithHashesMatchesSolve(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle hashes match")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}

	want, err := Solve(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	got, _, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}
	if got == nil || want == nil {
		t.Fatal("expected non-nil solutions")
	}
	if *got != *want {
		t.Fatalf("SolveWithHashes solution %+v != Solve %+v", *got, *want)
	}
}

func TestSolveContextWithHashesCanceled(t *testing.T) {
	th, err := FromBits(20)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sol, h, err := SolveContextWithHashes(ctx, []byte("equix-cgo/puzzle hashes cancel"), th, nonceStart)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if sol != nil {
		t.Fatalf("solution = %+v, want nil", sol)
	}
	if h != (equix.Hashes{}) {
		t.Fatalf("hashes = %v, want zero", h)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `CGO_ENABLED=1 go test ./puzz -run 'TestSolveWithHashes$|TestSolveWithHashesMatchesSolve|TestSolveContextWithHashesCanceled' -count=1`

Expected: 编译失败，提示 `undefined: SolveWithHashes`（以及 `SolveContextWithHashes`）。

- [ ] **Step 3: 编写最小实现**

在 `puzz/puzzle.go` 的 `SolveContext` 函数之后、`Verify` 之前插入。`Solve` / `SolveContext` 一行都不改。

```go
// SolveWithHashes 与 Solve 相同，额外返回命中解对应的 8 个 HashWX 哈希。
func SolveWithHashes(challenge []byte, th Threshold, nonce uint64) (*Solution, equix.Hashes, error) {
	return SolveContextWithHashes(context.Background(), challenge, th, nonce)
}

// SolveContextWithHashes 是带取消的 SolveWithHashes。取消与错误时解为 nil、哈希为零值。
func SolveContextWithHashes(ctx context.Context, challenge []byte, th Threshold, nonce uint64) (*Solution, equix.Hashes, error) {
	var hash [32]byte
	var zero equix.Hashes
	for {
		if err := ctx.Err(); err != nil {
			return nil, zero, err
		}
		results, err := equix.SolveWithHashesAndNonce(challenge, nonce)
		if err != nil {
			return nil, zero, err
		}
		for _, r := range results {
			combinedHash(&hash, challenge, nonce, r.Solution)
			if th.hit(&hash) {
				return &Solution{Nonce: nonce, Solution: r.Solution}, r.Hashes, nil
			}
		}
		nonce += nonceStep
	}
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `CGO_ENABLED=1 go test ./puzz -run 'TestSolveWithHashes$|TestSolveWithHashesMatchesSolve|TestSolveContextWithHashesCanceled|TestSolveAndVerify' -count=1`

Expected: 全部 PASS。`TestSolveAndVerify` 仍走无哈希路径，确认未被改坏。

- [ ] **Step 5: Commit**

```bash
git add puzz/puzzle.go puzz/puzzle_test.go
git commit -m "$(cat <<'EOF'
为 puzz 求解补上 HashWX 多返回值。

EOF
)"
```

---

### Task 2: VerifyWithHashes 与文档

**Files:**
- Modify: `puzz/puzzle.go`（在 `Verify` 之后追加 `VerifyWithHashes`）
- Modify: `puzz/puzzle_test.go`（追加校验测试）
- Modify: `puzz/doc.go`
- Modify: `puzz/README.md`（在「验证成本」一节之后插入「带 HashWX 哈希」，不改 24 字节编码一节）

**Interfaces:**
- Consumes: Task 1 的 `SolveWithHashes`；现有 `combinedHash` / `hit`；`equix.VerifyWithHashesAndNonce`
- Produces: `func VerifyWithHashes(challenge []byte, th Threshold, sol *Solution) (equix.Hashes, bool)`

- [ ] **Step 1: 编写失败的测试**

在 `puzz/puzzle_test.go` 的 `TestSolveContextWithHashesCanceled` 之后插入：

```go
func TestVerifyWithHashes(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle verify hashes")
	th, err := FromProbability(0.1)
	if err != nil {
		t.Fatal(err)
	}

	sol, want, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}

	got, ok := VerifyWithHashes(challenge, th, sol)
	if !ok {
		t.Fatal("VerifyWithHashes rejected a freshly solved puzzle")
	}
	if got != want {
		t.Fatalf("hashes mismatch: got %v want %v", got, want)
	}

	// 错误 nonce / 篡改解 / 错误 challenge 必须失败。
	badNonce := *sol
	badNonce.Nonce++
	if _, ok := VerifyWithHashes(challenge, th, &badNonce); ok {
		t.Fatal("VerifyWithHashes accepted a mutated nonce")
	}
	tampered := *sol
	tampered.Solution[0] ^= 1
	if _, ok := VerifyWithHashes(challenge, th, &tampered); ok {
		t.Fatal("VerifyWithHashes accepted a tampered solution")
	}
	if _, ok := VerifyWithHashes([]byte("equix-cgo/puzzle verify hashes!"), th, sol); ok {
		t.Fatal("VerifyWithHashes accepted a mismatched challenge")
	}

	// nil 提交必须失败而非 panic，且哈希为零值。
	h, ok := VerifyWithHashes(challenge, th, nil)
	if ok {
		t.Fatal("VerifyWithHashes accepted a nil solution")
	}
	if h != (equix.Hashes{}) {
		t.Fatalf("nil solution hashes = %v, want zero", h)
	}
}

func TestVerifyWithHashesStillReturnsOnOrder(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle verify hashes order")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}
	sol, _, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}

	bad := *sol
	bad.Solution[0], bad.Solution[1] = bad.Solution[1], bad.Solution[0]

	got, ok := VerifyWithHashes(challenge, th, &bad)
	if ok {
		t.Fatal("VerifyWithHashes accepted a swapped-index solution")
	}
	want, err := equix.VerifyWithHashesAndNonce(challenge, bad.Nonce, bad.Solution)
	if !errors.Is(err, equix.ErrOrder) {
		t.Fatalf("equix.VerifyWithHashesAndNonce: %v, want ErrOrder", err)
	}
	if got != want {
		t.Fatalf("hashes mismatch: got %v want %v", got, want)
	}
	if got == (equix.Hashes{}) {
		t.Fatal("hashes must still be populated")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `CGO_ENABLED=1 go test ./puzz -run 'TestVerifyWithHashes$|TestVerifyWithHashesStillReturnsOnOrder' -count=1`

Expected: 编译失败，提示 `undefined: VerifyWithHashes`。

- [ ] **Step 3: 编写最小实现**

在 `puzz/puzzle.go` 的 `Verify` 函数之后追加。结构校验失败时主包仍返回已算出的哈希，直接交出即可，不必用 `errors.Is` 区分错误类型；分配失败等内部错误时主包返回零值哈希。

```go
// VerifyWithHashes 与 Verify 相同，额外返回该解的 8 个 HashWX 哈希。
// 阈值未过或 sol 为 nil 时返回零值哈希和 false，不调用 Equi-X。
// 阈值通过但 Equi-X 失败时仍返回已算出的哈希和 false。
func VerifyWithHashes(challenge []byte, th Threshold, sol *Solution) (equix.Hashes, bool) {
	var zero equix.Hashes
	if sol == nil {
		return zero, false
	}
	var hash [32]byte
	combinedHash(&hash, challenge, sol.Nonce, sol.Solution)
	if !th.hit(&hash) {
		return zero, false
	}
	h, err := equix.VerifyWithHashesAndNonce(challenge, sol.Nonce, sol.Solution)
	return h, err == nil
}
```

- [ ] **Step 4: 更新文档**

`puzz/doc.go` 在 `[Verify]` 那段之后插入：

```go
// 需要同时得到解对应的 8 个 HashWX 哈希时，使用 [SolveWithHashes]、
// [SolveContextWithHashes] 与 [VerifyWithHashes]。哈希类型为
// [github.com/cxio/equix-cgo.Hashes]，作为多返回值，不进入 Solution 编码。
```

`puzz/README.md` 在「## 验证成本」一节之后、「## 实测」之前插入：

```markdown
## 带 HashWX 哈希

需要解对应的 8 个 HashWX 输出时，使用 `SolveWithHashes` / `VerifyWithHashes`。哈希类型为 `equix.Hashes`，作为多返回值，不进入 `Solution` 的 24 字节编码。

```go
sol, h, err := puzz.SolveWithHashes(challenge, th, 13)
if err != nil {
    panic(err)
}
got, ok := puzz.VerifyWithHashes(challenge, th, sol)
// ok == true 且 got == h；equix.VerifyHashes(h) == nil
_ = got
```

`VerifyWithHashes` 与 `Verify` 一样先做 SHA-256 门槛：未过或 `sol == nil` 时返回零值哈希和 `false`，不计算 HashWX。门槛通过但 Equi-X 结构校验失败时，仍返回已算出的哈希和 `false`。需要限时或取消时使用 `SolveContextWithHashes`。
```

- [ ] **Step 5: 运行测试验证通过**

Run: `CGO_ENABLED=1 go vet ./... && CGO_ENABLED=1 go test ./puzz -count=1`

Expected: `go vet` 无输出；`go test ./puzz` 全部 PASS。计时类 `TestSolveCost` 仍走无哈希路径。

- [ ] **Step 6: Commit**

```bash
git add puzz/puzzle.go puzz/puzzle_test.go puzz/doc.go puzz/README.md
git commit -m "$(cat <<'EOF'
为 puzz 校验补上 HashWX 多返回值。

EOF
)"
```
