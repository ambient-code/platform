---
type: correction
date: 2026-04-21T18:34:42Z
author: system-serviceaccount-ambient-code-test-user
title: "Session data uses K8s CRDs not PostgreSQL (demo mo8ypsqr)"
---

The Ambient Code Platform stores session data in Kubernetes Custom Resources (AgenticSession CRDs), not in PostgreSQL. The backend creates the CR, the operator watches it and creates a K8s Job to run the agent. PostgreSQL is only used by the ambient-api-server component for its own data model.
