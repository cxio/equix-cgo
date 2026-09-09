# puzz nonce 搜索步进参数 实现 Plan

> **给 agentic workers：** 必需子技能：使用 subagent-driven-development（推荐）或 executing-plans 逐任务实现此 plan。步骤使用复选框（`- [ ]`）语法进行跟踪。

**Goal:** 为 `puzz` 四个 nonce 搜索入口增加 `step uint64` 参数；`0` 使用默认 `nonceStep`（`0x26f5`）。

**Architecture:** 在 `SolveContext` / `SolveContextWithHashes` 进入循环前把 `step` 解析为有效步进（`0 → nonceStep`），循环内 `nonce += step`。`Solve` / `SolveWithHashes` 把 `step` 原样下传。哈希路径仍走 `equix.SolveWithHashesAndNonce`，无哈希路径仍走 `equix.SolveWithNonce`。

**Tech Stack:** Go 1.27、cgo、模块 `github.com/cxio/equix-cgo`（puzz 只调用主包，不 `import "C"`）。

## 全局约束

- 模块路径：`github.com/cxio/equix-cgo`
- 注释、文档、commit message 用简体中文；错误消息用英文
- 四个搜索入口统一加最后参数 `step uint64`；不保留旧三参数签名
- `step == 0` 解析为未导出的 `nonceStep`；非 0 原样使用；不因此返回 `error`
- 不导出 `nonceStep` / `NonceStep`
- 不改 `Try` / `Accept` / `Verify` / `VerifyWithHashes`
- 不改 `puzz.Solution`、24 字节编解码、哈希公式、取消语义
- 不把无哈希路径改走 `SolveWithHashesAndNonce`
- 不修改 `third_party/`、`internal/native/` 或主包公开 API
- 不改根 `README.md`
- `CGO_ENABLED=1`；验证命令：`go vet ./... && go test ./...`

## 文件结构

| 路径 | 职责 |
| --- | --- |
| `puzz/puzzle.go` | 四个搜索入口签名；私有 `effectiveStep`；两处循环用解析后的步进 |
| `puzz/puzzle_test.go` | 现有调用补末参 `0`；追加步进用例 |
| `puzz/doc.go` | 包注释：默认步进可被 `step` 覆盖 |
| `puzz/README.md` | 示例末参 `0`；步进与多 worker 说明 |

不新建文件。权威规格：`docs/specs/2026-09-09-puzz-nonce-step-design.md`。

---

### Task 1: 测试与四参数搜索 API

**Files:**
- Modify: `puzz/puzzle.go`（`effectiveStep`；`Solve` / `SolveContext` / `SolveWithHashes` / `SolveContextWithHashes`）
- Test: `puzz/puzzle_test.go`（现有 `Solve*` 调用补 `0`；在 `TestSolveContext` 之前追加步进用例）

**Interfaces:**
- Consumes: 现有 `nonceStep`、`Threshold`、`equix.SolveWithNonce`、`equix.SolveWithHashesAndNonce`
- Produces:
  - `func effectiveStep(step uint64) uint64`（未导出）
  - `func Solve(challenge []byte, th Threshold, nonce, step uint64) (*Solution, error)`
  - `func SolveContext(ctx context.Context, challenge []byte, th Threshold, nonce, step uint64) (*Solution, error)`
  - `func SolveWithHashes(challenge []byte, th Threshold, nonce, step uint64) (*Solution, equix.Hashes, error)`
  - `func SolveContextWithHashes(ctx context.Context, challenge []byte, th Threshold, nonce, step uint64) (*Solution, equix.Hashes, error)`

- [ ] **Step 1: 编写失败的测试**

把 `puzz/puzzle_test.go` 里所有 `Solve` / `SolveContext` / `SolveWithHashes` / `SolveContextWithHashes` 调用补上末参 `0`。完整列表：

- `TestAcceptMatchesVerifyNonce`：`Solve(challenge, th, nonceStart, 0)`
- `TestSolveAndVerify`：`Solve(challenge, th, nonceStart, 0)`
- `TestSolveWithHashes`：`SolveWithHashes(challenge, th, nonceStart, 0)`
- `TestSolveWithHashesMatchesSolve`：`Solve(..., 0)` 与 `SolveWithHashes(..., 0)`
- `TestSolveContextWithHashesCanceled`：`SolveContextWithHashes(ctx, ..., nonceStart, 0)`
- `TestVerifyWithHashes`：`SolveWithHashes(..., 0)`
- `TestVerifyWithHashesStillReturnsOnOrder`：`SolveWithHashes(..., 0)`
- `TestSolveContext`：两处 `SolveContext(..., nonceStart, 0)`
- `TestConcurrent`：`SolveContext(ctx, challenge, th, nonceStart+uint64(g), 0)`
- `TestSolveCost`：`Solve(challenge, th, nonceStart, 0)`（轮数计算仍用 `nonceStep`）
- `BenchmarkSolve`：`Solve(challenge, th, nonceStart, 0)`

在 `TestSolveContext` 之前插入：

```go
func TestSolveStepDefault(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle step default")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}

	a, err := Solve(challenge, th, nonceStart, 0)
	if err != nil {
		t.Fatalf("Solve step=0: %v", err)
	}
	b, err := Solve(challenge, th, nonceStart, nonceStep)
	if err != nil {
		t.Fatalf("Solve step=nonceStep: %v", err)
	}
	if a == nil || b == nil || *a != *b {
		t.Fatalf("Solve step 0 = %+v, nonceStep = %+v", a, b)
	}

	ha, _, err := SolveWithHashes(challenge, th, nonceStart, 0)
	if err != nil {
		t.Fatalf("SolveWithHashes step=0: %v", err)
	}
	hb, _, err := SolveWithHashes(challenge, th, nonceStart, nonceStep)
	if err != nil {
		t.Fatalf("SolveWithHashes step=nonceStep: %v", err)
	}
	if ha == nil || hb == nil || *ha != *hb {
		t.Fatalf("SolveWithHashes step 0 = %+v, nonceStep = %+v", ha, hb)
	}
}

func TestSolveStepResidue(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle step residue")
	th, err := FromBits(DefaultBits)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []uint64{1, 7} {
		sol, err := Solve(challenge, th, nonceStart, step)
		if err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		if sol == nil {
			t.Fatalf("step %d: nil solution", step)
		}
		if (sol.Nonce-nonceStart)%step != 0 {
			t.Fatalf("step %d: nonce %d not on sequence from %d", step, sol.Nonce, nonceStart)
		}
		if !Verify(challenge, th, sol) {
			t.Fatalf("step %d: Verify rejected", step)
		}
	}
}

func TestSolveWithHashesStepMatchesSolve(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle step hashes match")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}
	const step uint64 = 7
	want, err := Solve(challenge, th, nonceStart, step)
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	got, _, err := SolveWithHashes(challenge, th, nonceStart, step)
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

func TestSolveContextStepCanceled(t *testing.T) {
	th, err := FromBits(20)
	if err != nil {
		t.Fatal(err)
	}
	challenge := []byte("equix-cgo/puzzle step cancel")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	const step uint64 = 7
	if _, err := SolveContext(ctx, challenge, th, nonceStart, step); !errors.Is(err, context.Canceled) {
		t.Fatalf("SolveContext: %v, want context.Canceled", err)
	}
	sol, h, err := SolveContextWithHashes(ctx, challenge, th, nonceStart, step)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SolveContextWithHashes: %v, want context.Canceled", err)
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

Run: `go test ./puzz -count=1 -run 'TestSolveStep|TestSolveAndVerify'`

Expected: 编译失败，提示 `Solve` / `SolveContext` / `SolveWithHashes` 参数太多（旧签名仍是 3 个搜索参数，不含 `step`）。

- [ ] **Step 3: 编写最小实现**

在 `puzz/puzzle.go` 的 `nonceStep` 常量之后追加：

```go
func effectiveStep(step uint64) uint64 {
	if step == 0 {
		return nonceStep
	}
	return step
}
```

将四个函数改为：

```go
// Solve 从 nonce 起步搜索命中难度的解：每个 nonce 调用一次 Equi-X 求解，
// 对每个候选解做阈值判定，未命中则 nonce 前进 step 后重试。
// step 为 0 时使用默认 nonceStep（0x26f5）。
// 搜索没有天然终点（阈值越低期望耗时越长），需要限时或取消机制时可用 SolveContext。
func Solve(challenge []byte, th Threshold, nonce, step uint64) (*Solution, error) {
	return SolveContext(context.Background(), challenge, th, nonce, step)
}

// SolveContext 是带取消的 Solve：进入时及每轮求解（约数十毫秒）之间检查
// ctx，取消时返回 ctx.Err()（context.Canceled 或 context.DeadlineExceeded）。
// 取消最多延迟一轮生效；若当前轮已产出命中解，仍优先返回该有效结果。
func SolveContext(ctx context.Context, challenge []byte, th Threshold, nonce, step uint64) (*Solution, error) {
	step = effectiveStep(step)
	var hash [32]byte
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sols, err := equix.SolveWithNonce(challenge, nonce)
		if err != nil {
			return nil, err
		}
		for _, sol := range sols {
			combinedHash(&hash, challenge, nonce, sol)
			if th.hit(&hash) {
				return &Solution{Nonce: nonce, Solution: sol}, nil
			}
		}
		nonce += step
	}
}

// SolveWithHashes 与 Solve 相同，额外返回命中解对应的 8 个 HashWX 哈希。
func SolveWithHashes(challenge []byte, th Threshold, nonce, step uint64) (*Solution, equix.Hashes, error) {
	return SolveContextWithHashes(context.Background(), challenge, th, nonce, step)
}

// SolveContextWithHashes 是带取消的 SolveWithHashes。取消与错误时解为 nil、哈希为零值。
func SolveContextWithHashes(ctx context.Context, challenge []byte, th Threshold, nonce, step uint64) (*Solution, equix.Hashes, error) {
	step = effectiveStep(step)
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
		nonce += step
	}
}
```

不得把无哈希循环改成调用 `SolveWithHashesAndNonce`。`effectiveStep` 只在进入循环前调用一次（通过给参数赋值）。

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./puzz -count=1`

Expected: PASS（含 `TestSolveCost`，约数秒到十余秒）。

- [ ] **Step 5: Commit**

```bash
git add puzz/puzzle.go puzz/puzzle_test.go
git commit -m "$(cat <<'EOF'
为 puzz 带 nonce 的搜索入口增加步进参数。

EOF
)"
```

---

### Task 2: 文档

**Files:**
- Modify: `puzz/doc.go`
- Modify: `puzz/README.md`（示例调用、难度预算与多 worker 一节）

**Interfaces:**
- Consumes: Task 1 的四参数签名与 `step==0` 语义
- Produces: 与实现一致的包注释和 README

- [ ] **Step 1: 更新 puzz/doc.go**

将「[Solve] 从指定 nonce 起步、以素数 0x26f5 步进反复求解」改为说明默认步进及 `step` 覆盖：

```go
// [Solve] 从指定 nonce 起步、默认以素数 0x26f5 步进反复求解，直到命中难度。
// 最后一个参数 step 为 0 时使用该默认值，正值则作为自定义步进。
// 搜索没有天然终点，需要限时或取消时使用 [SolveContext]。
```

其余段落保持不变。

- [ ] **Step 2: 更新 puzz/README.md**

1. 用法示例：`puzz.Solve(challenge, th, 13)` → `puzz.Solve(challenge, th, 13, 0)`。
2. 取消示例：`puzz.SolveContext(ctx, challenge, th, 13)` → 末参 `0`。
3. HashWX 示例：`puzz.SolveWithHashes(challenge, th, 13)` → 末参 `0`。
4. 「难度预算的口径」中步进段落改为：

```markdown
nonce 默认步进为素数 `0x26f5`（9973），`Solve*` 的 `step` 参数为 `0` 时使用该值，正值则覆盖。奇素数步进可避免多 worker 的搜索序列与 2 的幂步进对齐；多 worker 并行分片时，使用相同有效步进的各起点应避免相差该步进的整数倍，否则搜索序列完全重叠。
```

- [ ] **Step 3: 运行测试确认文档未破坏构建**

Run: `go vet ./... && go test ./...`

Expected: PASS。计时类测试约 10s。

- [ ] **Step 4: Commit**

```bash
git add puzz/doc.go puzz/README.md
git commit -m "$(cat <<'EOF'
同步 puzz 文档中的 nonce 步进参数。

EOF
)"
```

---

## Spec 覆盖

| Spec 需求 | 任务 |
| --- | --- |
| 四入口加 `step uint64` | Task 1 |
| `0 → nonceStep`，正值原样，进入循环前解析一次 | Task 1 `effectiveStep` |
| 无哈希仍走 `SolveWithNonce` | Task 1 |
| 不导出 `nonceStep` | Task 1（保持未导出） |
| 不改 Try/Accept/Verify* | 无对应改动 |
| 现有用例补 `0` | Task 1 Step 1 |
| 默认等价 / 取模 / 两路径一致 / 取消 | Task 1 四个新测试 |
| README / doc.go | Task 2 |
| 不改根 README | 无对应改动 |
