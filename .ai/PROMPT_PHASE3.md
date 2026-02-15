# PROMPT_PHASE3.md — huectl: richer `lights list` output (groups + state)

You are an autonomous coding agent working in this repo. Implement Phase 3 improvements to `huectl lights list` output.

Follow repo rules in `AGENTS.md` and `.ai/AGENTS.md`:
- Use `jj` for all VCS actions.
- Make local commits for meaningful milestones.
- Do NOT push to origin unless I explicitly ask.

## Goal
Enhance `huectl lights list` to optionally include **Room/Zone (group)** information and a few useful state fields, without breaking existing scripts.

## Requirements

### Output & flags
- Keep current default output exactly as-is:
  - Default columns (CSV): `id,name,on`
  - Default (non-CSV): same info you currently show
- Add flags:
  - `--with-group`  
    Adds group metadata columns/fields:
    - `group` (grouped_light name)
    - `group_type` (room/zone if available; otherwise the API’s type string)
  - `--with-state`  
    Adds state columns/fields (only include what the API provides reliably):
    - `reachable` (or `available`) boolean if Hue exposes it
    - `bri` brightness as `0-100` if available (blank if unknown)
  - `--wide`  
    Equivalent to `--with-group --with-state`
- For CSV output:
  - Preserve column order:
    - default: `id,name,on`
    - with-group: `id,name,group,group_type,on`
    - with-state: `id,name,on,reachable,bri`
    - wide: `id,name,group,group_type,on,reachable,bri`
- For non-CSV output:
  - Add extra columns/fields only when the flags are set.

### Semantics
- “Group” in this context means Hue API v2 **`grouped_light`** resources (covers rooms and zones).
- Build a mapping from **light ID -> grouped_light name/type**.
  - If multiple groups match a light (unlikely), pick the first deterministically and document behavior.
  - If no group found, leave `group` and `group_type` blank.
- Performance:
  - Avoid per-light API calls. Fetch grouped_light resources once and build the mapping.
  - Keep API calls minimal (ideally: lights + grouped_lights; and only extra calls if required for reachability/brightness).

### Testing (“closing the loop”)
- Add/update tests so `go test ./...` is green.
- Use `httptest.Server` to mock Hue v2 endpoints.
- Add tests for:
  - CSV header/column sets for each flag combination.
  - Correct group mapping for at least two lights in different groups.
  - Behavior when a light has no group.
  - Deterministic behavior if multiple groups reference a light (test fixture).

### Docs
- Update `README.md`:
  - Document `--with-group`, `--with-state`, `--wide`
  - Show examples for both normal and `--csv` output.
- If you added any new internal types/endpoints, update `PLAN.md` notes if needed.

## Implementation guidance (be pragmatic)
- Prefer small, composable functions:
  - fetch grouped_lights
  - build light->group map
  - render output with selected columns
- Keep changes localized to the `lights list` command + Hue client, but refactor lightly if it reduces duplication.

## Done criteria
- `make test`, `make lint`, and `make build` succeed.
- Default output unchanged.
- New flags work as specified.
- Tests cover the new behavior.
- Make at least 1–2 logical `jj` commits with clear messages (no pushing).

Start now.
