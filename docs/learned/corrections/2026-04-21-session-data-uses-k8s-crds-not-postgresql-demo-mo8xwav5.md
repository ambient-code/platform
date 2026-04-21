---
type: correction
date: 2026-04-21T18:12:01Z
author: system-serviceaccount-ambient-code-test-user
title: "Session data uses K8s CRDs not PostgreSQL (demo mo8xwav5)"
---

The Ambient Code Platform stores session data in Kubernetes Custom Resources (AgenticSession CRDs), not in PostgreSQL. The backend creates the CR, the operator watches it and creates a K8s Job to run the agent. PostgreSQL is only used by the ambient-api-server component for its own data model.

Source: User corrections during session — agent incorrectly described the storage layer.
