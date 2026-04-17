---
title: Mounting a Custom CA Bundle
description: How to configure Ambient to trust certificates from a private or corporate CA
---

By default, Ambient components only trust certificates signed by the public CAs baked into the base container image. If your deployment connects to services that use a private or corporate CA — for example, an internal Git host — you need to provide that CA bundle to the relevant components.

## How it works

Ambient reads the system CA bundle at `/etc/pki/tls/certs/ca-bundle.crt` inside each container. Replacing or augmenting that file with your CA certificates causes all outbound TLS connections to trust those CAs automatically — no code changes or custom environment variables needed.

## On OpenShift: CA bundle injection

OpenShift can automatically populate a ConfigMap with the cluster's full trusted CA bundle (including any custom CAs configured via the cluster proxy). To use this:

**1. Create the ConfigMap in each relevant namespace**

Apply the following manifest, changing `namespace` to match your deployment:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: trusted-ca-bundle
  namespace: <your-namespace>
  labels:
    config.openshift.io/inject-trusted-cabundle: "true"
data: {}
```

Leave `data: {}` empty — OpenShift's CA bundle injector will populate the `ca-bundle.crt` key automatically.

**2. Mount it into the backend Deployment**

Patch your backend `Deployment` to mount the ConfigMap over the system CA path:

```yaml
spec:
  template:
    spec:
      volumes:
        - name: trusted-ca-bundle
          configMap:
            name: trusted-ca-bundle
      containers:
        - name: backend-api
          volumeMounts:
            - name: trusted-ca-bundle
              mountPath: /etc/pki/tls/certs/ca-bundle.crt
              subPath: ca-bundle.crt
              readOnly: true
```

**3. Verify**

After the pod restarts, confirm it can connect to your internal service:

```bash
kubectl exec deployment/backend-api -- curl -I https://your-internal-host
```

## On other Kubernetes distributions

If you are not using OpenShift's CA injector, create the ConfigMap yourself with the PEM-encoded CA certificate(s):

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: trusted-ca-bundle
  namespace: <your-namespace>
data:
  ca-bundle.crt: |
    -----BEGIN CERTIFICATE-----
    <your CA certificate here>
    -----END CERTIFICATE-----
```

Then apply the same volume mount as above.

## Current support status

| Component | Custom CA support |
|-----------|-------------------|
| `backend-api` | Supported — mount the ConfigMap as shown above |
| Runner pods | Pending — see [#1247](https://github.com/ambient-code/platform/issues/1247) and [#1038](https://github.com/ambient-code/platform/issues/1038) |

Runner pods are created dynamically by the agentic-operator for each session. Until #1247 is resolved, the operator does not mount the ConfigMap into runner pods, so runners cannot connect to services that require a custom CA.
