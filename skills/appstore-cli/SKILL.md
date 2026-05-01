---
name: appstore-cli
description: Use this skill when working with the local `appstore` CLI for direct App Store API reads (`search`, `hints`, `app-details`) using access-token profiles, multi-profile context switching, and agent-safe request patterns.
---

# App Store CLI

Use this skill for repository-local App Store CLI work.

## Workflow

1. Resolve the target access-token profile first.
2. Verify the active profile before auth-required reads.
3. Use direct typed commands (`search`, `hints`, `app-details`) rather than raw curl where possible.
4. Prefer stable, explicit flags (`storefront`, `platform`, `language`) for reproducible outputs.

## Profile Resolution

- Inspect profiles with `appstore auth list` or `appstore auth show`.
- Select the active profile with `appstore auth use <name>`.
- Override profile per command with `--profile <name>`.
- Add a profile with `appstore auth add <name> --access-token <token>`.
- Never print raw access tokens in normal output.

## Command Pattern

- Use `appstore search --keyword <term> --storefront <code> --platform <platform>` for app search.
- Use `appstore app-details --app-id <id> --storefront <code> --language <locale> --platform <platform>` for app detail lookups.
- Use `appstore hints --term <term>` for autocomplete.

## Hints Guardrail

- `hints` is intentionally sent without Authorization by default to reduce bearer-based throttling.
- Use `appstore hints --with-auth --profile <name> ...` only when explicitly needed.
- If `hints` returns 429, retry without `--with-auth` and reduce request burst.

## Error Handling Pattern

- 401 usually means invalid or expired token in the selected profile.
- 403 usually means token scope/account is not allowed for that endpoint.
- 429 means upstream throttling; back off and retry with lower frequency.

## Agent Safety

- Keep requests read-only and profile-scoped.
- Do not store or echo sensitive tokens in logs, markdown, or commit history.
