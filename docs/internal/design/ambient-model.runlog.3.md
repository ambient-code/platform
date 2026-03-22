# Ambient Model: Run Log 3 — 2026-03-21

**Status:** Spec updated. Wave 5 (CLI) committed (`2c56b91a`). Waves 2–5 complete. Beginning new spec-to-code reconciliation cycle.

---

## Spec Changes This Cycle

- `Agent.owner_user_id` removed — ownership is now expressed via RoleBinding (`scope=agent`, `scope_id=agent_id`), not a direct FK on the Agent struct
- `Project.display_name` removed — redundant; `name` is the identity; display formatting is a UI concern

---

## Gap Table (opening state)

```
ENTITY/ROUTE                            COMPONENT  STATUS    GAP
Agent.owner_user_id removal            API        missing   remove field from openapi.agents.yaml
Agent.owner_user_id removal            BE         missing   remove from model.go, migration drop column
Agent.owner_user_id removal            SDK        missing   regen after API change
Project.display_name removal           API        missing   remove field from openapi.projects.yaml
Project.display_name removal           BE         missing   remove from model.go, migration drop column
Project.display_name removal           SDK        missing   regen after API change
Project.display_name removal           CLI        missing   remove --display-name flag from create project
Agent struct drift (model.go)          BE         missing   model has ProjectId, ParentAgentId, LlmModel, etc. not in spec
```

Notable model drift in `plugins/agents/model.go`: fields present in code but absent from spec at the time of audit — `ProjectId`, `ParentAgentId`, `OwnerUserId`, `DisplayName`, `Description`, `RepoUrl`, `WorkflowId`, `LlmModel`, `LlmTemperature`, `LlmMaxTokens`, `BotAccountName`, `ResourceOverrides`, `EnvironmentVariables`, `CurrentSessionId`.

---

## Stopped At

Spec commit complete. Awaiting Wave 2 (API openapi changes) before code changes proceed.
