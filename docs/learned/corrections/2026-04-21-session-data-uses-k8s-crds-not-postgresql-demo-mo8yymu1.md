---
type: correction
date: 2026-04-21T18:41:31Z
author: system-serviceaccount-ambient-code-test-user
title: "Session data uses K8s CRDs not PostgreSQL (demo mo8yymu1)"
---

Session data uses K8s CRDs not PostgreSQL. The operator creates Jobs from CRDs. PostgreSQL is only used by the api-server.
