# ADR-0007: CI/CD Deployment Strategy for Firewalled OpenShift Clusters

**Date:** 2026-03-04
**Authors:** Ken Dreyer (with Gemini 3 Pro)
**Deciders:** Platform Team

## Context

We currently host our app on public GitHub and deploy to a public ROSA cluster using GitHub Actions and the `oc` CLI. We are moving to a firewalled OpenShift cluster (MP+), which breaks our current public GitHub runner workflow.

We need to preserve two properties:

1. **Immediacy:** When a developer merges code to `main`, we must deploy immediately without waiting for a polling cycle.
2. **Visibility:** Developers need direct access to deployment logs to troubleshoot their own issues without Ops help.

## Decision

We will run self-hosted GitHub Actions runners inside our firewalled OpenShift cluster. The runners make outbound connections to GitHub, pick up jobs, and execute them locally. Because they live inside the cluster and firewall, they can talk directly to the OpenShift API via `oc`.

We are rolling this out in two phases.

This week I have already moved forward with trialing Phase 1.

### Phase 1: Standalone Runner

Deploy a standalone GitHub Actions runner as a regular `Deployment` — no CRDs, no `ClusterRoles`, no operator.

**How it works:**

* A `Deployment` with `replicas: 1` runs the [GitHub Actions runner agent](https://github.com/actions/runner/pkgs/container/actions-runner).
* At startup, the runner uses GitHub App credentials (App ID + private key) to generate a short-lived registration token and register itself with GitHub.
* The runner's `ServiceAccount` only needs the permissions our CI jobs require (e.g. `oc apply` to target namespaces). It does not need any cluster-level permissions.

**Pros:**

* No CRDs or cluster-level RBAC — deploys with namespace-scoped permissions only
* No IT approval needed — can deploy immediately
* Architecturally simple — a single long-running pod
* Identical developer experience — jobs appear in the GitHub Actions UI the same way
* Serialized deploys — jobs run one at a time, so concurrent merges cannot trample each other in prod

**Cons:**

* No auto-scaling — the runner pod is always running regardless of job queue depth
* Single point of failure — if the pod crashes, jobs silently queue instead of running

### Phase 2: Actions Runner Controller (ARC)

If IT approves CRD installation in preprod/prod (requires a ServiceNow ticket), we can upgrade to the [Actions Runner Controller (ARC)](https://github.com/actions/actions-runner-controller) operator. ARC dynamically creates and destroys runner pods based on the job queue.

**What ARC adds over the standalone runner:**

* Auto-scaling — runner pods scale up and down based on demand, saving compute resources
* Multi-runner — can run multiple jobs concurrently. *We would need to investigate how to prevent concurrent deploy jobs from trampling each other in prod.*

**What ARC requires:**

* Custom Resource Definitions installed in the cluster (IT approval)
* Cluster-level RBAC for the operator
* Ongoing maintenance and patching of the ARC operator

Phase 2 depends on IT approving CRDs.
 * If IT *does not* approve, we remain on Phase 1.
 * If IT *does approve*, we will retire standalone GH Action Runner `Deployment` and replace it with ARC.

## Considered Options

### Option 1: Self-hosted GitHub Actions runners (standalone or ARC) — chosen

See Phase 1 and Phase 2 above.

### Option 2: OpenShift GitOps (ArgoCD) with an Ingress Tunnel — rejected

Rejected because it forces developers to learn a new UI (ArgoCD) to view their logs. It also requires punching a hole in the firewall for the webhook, which is needlessly complex and less secure compared to the outbound-only model.

### Option 3: VPN/SSH/Network Overlay with GitHub Actions — rejected

Rejected due to operational complexity and security concerns with maintaining persistent network tunnels into the firewalled cluster.

## Consequences

**Positive:**

* Developers keep their existing GitHub Actions workflow and can debug deployments without Ops
* No inbound firewall ports — aligns with Infosec standards
* Instant job pickup preserves deployment velocity
* Phase 1 can deploy immediately with no IT dependencies

**Negative:**

* We must pay for and manage the compute resources for the runner(s)
* Dependency on GitHub App credentials with periodic rotation
* Phase 1 has no auto-scaling and no built-in redundancy

**Risks:**

* No alerting on standalone runner failure. If the pod crashes, deployments silently stop and GitHub Actions jobs queue instead of running. We need monitoring to detect this.
* IT may not approve CRDs for Phase 2, leaving us on Phase 1 permanently.
* Moving to Phase 2 introduces concurrent deploy jobs. We would need to investigate serialization or locking to prevent jobs from trampling each other in prod.

## Risks of Remaining on Phase 1

If we cannot move to Phase 2, the standalone runner carries ongoing operational risks:

* **Single point of failure.** One pod handles all CI jobs. If it crashes or is evicted, no jobs run until the pod restarts, without notifying a person or agentic process.
* **No concurrency.** Jobs run sequentially, which prevents deploy races but increases latency when multiple PRs merge quickly.
* **No auto-scaling.** The runner pod runs continuously regardless of load — wasting resources when idle, unable to scale during bursts.
* **Manual recovery.** If the runner loses its GitHub registration (e.g. after a credential rotation or a prolonged outage), someone must re-register it manually.
* **No built-in high availability.** Running multiple replicas of the standalone runner may cause conflicts with job pickup and GitHub registration. A high-availability solution would require further investigation.

## Concurrent Deploys

Phase 1 serializes deploys naturally — one runner, one job at a time. If we move to Phase 2 (ARC), concurrent runners could deploy conflicting changes simultaneously.

Before enabling concurrent runners, we must serialize deploy jobs to prevent parallel deploys from overwriting each other in prod. The GitHub Actions [`concurrency`](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/control-the-concurrency-of-workflows-and-jobs) key handles this:

```yaml
concurrency:
  group: deploy-prod
  cancel-in-progress: false
```

This makes deploys sequential again, even with multiple runners — negating most of ARC's benefit. It is a safe starting point, not a long-term solution. If we later need parallel deploys, we will need a broader strategy to prevent conflicts (e.g. environment locking, progressive rollouts). We have not yet scoped that work.

## References

* [Actions Runner Controller](https://github.com/actions/actions-runner-controller)
* [GitHub ARC authentication docs](https://docs.github.com/en/actions/tutorials/use-actions-runner-controller/authenticate-to-the-api#deploying-using-github-app-authentication)
* [GitHub Actions runner image](https://github.com/actions/runner/pkgs/container/actions-runner)
