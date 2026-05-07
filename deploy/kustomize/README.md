# Kubernetes deployment (Kustomize)

Shared **base**, environment **overlays**, and **secrets** with a Sealed Secrets workflow.

## Structure

```
deploy/kustomize/
├── base/                 # Namespace, PVC, Deployment, Service
├── overlays/
│   └── k3s/              # Example cluster overlay (ingress + sealed secrets)
└── secrets/              # Secret templates & sealing (see secrets/README.md)
```

## Deploying

```bash
# Preview manifests
kubectl kustomize deploy/kustomize/overlays/k3s

# Apply
kubectl apply -k deploy/kustomize/overlays/k3s
```

Requires [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) installed on the cluster so `SealedSecret` resources are unsealed into `gelift-tracker-secrets`.

## Secrets

MQTT credentials (and optional `DATABASE_URL`) live in a sealed secret checked in under `overlays/k3s/sealed-gelift-tracker-secrets.yaml`. To create or rotate secrets, follow **`secrets/README.md`** (fetch cert from `sealed-secrets.gewis.nl`, edit template, `kubeseal`, commit sealed YAML).

Non-sensitive environment (paths, ports) stays in **`base/deployment.yaml`**.
