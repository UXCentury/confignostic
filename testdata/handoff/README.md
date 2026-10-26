# Codex implementation → Cursor review fixture

This is a local acceptance fixture for A02 and later stages I–M. It contains no provider calls or simulated agent integration.

## Bounded task

Copy base/ into a disposable repository. Give only that candidate repository and the following requirement to the implementing agent:

> Update normalizeAlias so string aliases are trimmed and lowercase. Preserve TypeError for non-string input and RangeError for empty/whitespace-only input. Keep the export name and avoid dependencies. Modify only alias.mjs and, if useful, add tests in alias.test.mjs. Run the existing tests. Return the change and test result without publishing or pushing it.

The baseline already trims and validates input. It deliberately does not lowercase. Its original tests pass, while the external checker rejects it. The external checker belongs to the orchestrator/reviewer; the implementation task must not modify it.

No exact source-code solution is required: acceptance evaluates observable behavior. The documented fixture cases use ASCII aliases; broader identity rules belong to the stage C domain contract.

## Local walkthrough

From the Confignostic repository root:

```sh
make check-reference
candidate_dir="$(mktemp -d)"
cp -R testdata/handoff/base/. "$candidate_dir"
node --test "$candidate_dir/alias.test.mjs"
node scripts/check-reference.mjs "$candidate_dir"
```

The final command must exit 1 for the unchanged candidate. After the bounded implementation, both the original tests and that same external command must exit 0. Running the checker with no candidate argument exits 2.

Keep candidate directories disposable. The checker imports and executes candidate code; later real agent runs must use the isolated runner, not the control-plane process. These commands authorize only local fixture checks, not publication or external actions.

## Review handoff

Record the base revision/digest, candidate artifact digest, allowed file changes, implementation task/session identity and test evidence. Give Cursor a read-only copy of that exact candidate and the diff from the base.

The reviewer should check the requirements, rerun the original and external checks in an isolated workspace, and return a verdict, concrete findings and the reviewed artifact digest. It must not silently repair the candidate or review a moving working directory.

The handoff payload is an acceptance requirement, not an implemented API schema. Stage C will define its versioned representation; stage M will enforce its integrity.

## Acceptance matrix

| Case | Expected behavior |
| --- | --- |
| Original baseline tests | Pass; demonstrates a bounded missing behavior. |
| Unchanged candidate | External checker fails. |
| Correct lowercase change preserving validation | Original and external checks pass. |
| Wrong input type or empty alias becomes accepted | External checker fails. |
| Candidate changes outside the allowed files | Orchestration/review rejects the scope violation; checker alone is insufficient. |
| Output/base digest changes before review | Handoff rejected. |
| Agent/account/runner unavailable | Task reports unavailable or blocked, not success. |
| API restarts or acknowledgment is lost | Existing task is reconciled, never blindly duplicated. |
| Permission denied | Scoped action does not execute. |

Only the local behavioral rows are executable now. Provider execution, scope enforcement, digest-bound handoff, permissions and recovery remain planned.

## Fixture self-checks

reference.test.mjs verifies that the original tests pass and the external checker rejects the baseline. It creates a temporary copy with a known bounded change and checks that both suites accept it, then removes its own temporary directory. The source fixture stays unchanged. This gives CI a deterministic positive and negative oracle without real credentials or provider costs.
