# Vertex AI (ANTHROPIC_VERTEX_PROJECT_ID) on OpenShift

The production overlay uses Vertex AI by default (`USE_VERTEX=1`). You must set your GCP project ID and provide credentials.

## 1. Set your project ID and region

Patch the operator ConfigMap (replace with your values):

```bash
export KUBECONFIG=~/.kube/kubeconfig-noingress
oc patch configmap operator-config -n ambient-code --type merge -p '{
  "data": {
    "ANTHROPIC_VERTEX_PROJECT_ID": "YOUR_GCP_PROJECT_ID",
    "CLOUD_ML_REGION": "us-central1"
  }
}'
```

- **ANTHROPIC_VERTEX_PROJECT_ID**: Your Google Cloud project ID where Claude is enabled on Vertex AI.
- **CLOUD_ML_REGION**: Vertex AI region (e.g. `us-central1`, `europe-west1`, or `global` for some setups).

## 2. Create the GCP credentials secret

The operator and runner need a GCP service account key (or Application Default Credentials file) to call Vertex AI.

**Option A – Application Default Credentials (e.g. from your laptop):**

```bash
gcloud auth application-default login
oc create secret generic ambient-vertex -n ambient-code \
  --from-file=ambient-code-key.json="$HOME/.config/gcloud/application_default_credentials.json" \
  --dry-run=client -o yaml | oc apply -f -
```

**Option B – Service account key file:**

```bash
oc create secret generic ambient-vertex -n ambient-code \
  --from-file=ambient-code-key.json=/path/to/your-service-account-key.json
```

The key file must be the JSON for a service account that has Vertex AI (and optionally Model Garden) permissions.

## 3. Restart the operator

So it picks up the updated ConfigMap and uses the new project ID for new sessions:

```bash
oc rollout restart deployment/agentic-operator -n ambient-code
oc rollout status deployment/agentic-operator -n ambient-code --timeout=120s
```

## 4. Verify

- In the UI, create or open a session; it should use Vertex AI for that project.
- Check operator logs:  
  `oc logs -l app=agentic-operator -n ambient-code --tail=50 | grep -i vertex`

## Optional: change the default in the overlay

To bake your project ID into the overlay (e.g. for Git-managed deploys), edit:

- `overlays/production/operator-config-openshift.yaml`

Set `ANTHROPIC_VERTEX_PROJECT_ID` and `CLOUD_ML_REGION` to your values, then re-apply the overlay.

## Disable Vertex AI (use direct Anthropic API)

To use an Anthropic API key instead of Vertex:

```bash
oc patch configmap operator-config -n ambient-code --type merge -p '{"data":{"USE_VERTEX":"0"}}'
oc rollout restart deployment/agentic-operator -n ambient-code
```

Then configure `ANTHROPIC_API_KEY` in the UI (Settings → Runner Secrets) per project.
