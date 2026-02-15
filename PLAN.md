# Phase 2 Plan

## 1. Implement Hue API v2 client

- Base URL: `https://{bridge_host}/clip/v2`
- Authentication: `hue-application-key: <token>` header on every request.
- Endpoints to use:
  - List lights: `GET /resource/light`
  - Get light by ID (for toggle read/verification if needed): `GET /resource/light/{id}`
  - Update light state: `PUT /resource/light/{id}`

## 2. Data model and request building

- Parse Hue response envelope (`data`, `errors`) into typed structs.
- Treat Hue resource IDs as strings throughout.
- Build partial update payloads so only user-supplied fields are sent:
  - `on` / `off`
  - brightness (`dimming.brightness`)
  - color temperature (`color_temperature.mirek`)
  - XY color (`color.xy`)

## 3. CLI command implementation

- Implement:
  - `huectl lights list`
  - `huectl lights toggle --id <id>`
  - `huectl lights set --id <id> [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]`
- Validate flag combinations and ranges.
- Keep output concise and avoid leaking secrets.

## 4. Error handling and UX

- Surface bridge/API errors with actionable messages.
- Handle malformed JSON and network timeouts cleanly.
- Keep command exit codes non-zero on operational failures.

## 5. Testing strategy

- Use `httptest.Server` for all Hue API interactions (no real bridge/internet).
- Unit test request serialization and response parsing.
- Table-driven tests for CLI parsing and validation.
- Cases to include:
  - Successful list/toggle/set paths
  - Missing config/token
  - Partial updates include only specified fields
  - Hue API error envelope propagation
