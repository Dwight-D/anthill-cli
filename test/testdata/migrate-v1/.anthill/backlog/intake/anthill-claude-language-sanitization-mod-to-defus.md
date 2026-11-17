## Description

Add a `language-sanitization` feature module to the `anthill-claude` plugin
(`anthill-claude/hooks/features/`, registered from `hooks/register.tsx` like
`timestamps`).

Purpose: protect against false-positive model safeguards. Certain phrasings
make the safeguard classify an ordinary task as an AI-research / reasoning-
extraction task and the request is refused.
