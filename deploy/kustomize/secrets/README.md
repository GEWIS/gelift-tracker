# Sealed Secrets

Templates and sealed secrets for Kubernetes overlays.

## Workflow

1. Fetch the kubeseal public certificate (requires GEWIS network access):

   ```bash
   curl -s https://sealed-secrets.gewis.nl/v1/cert.pem > ../cert.pem
   ```

2. Copy a template and fill in real values:

   ```bash
   cp gelift-tracker-secrets.k3s.template.yaml gelift-tracker-secrets.k3s.yaml
   # Edit gelift-tracker-secrets.k3s.yaml with real values
   ```

3. Seal the secret (`namespace-wide` matches how SudoSOS seals backend env secrets):

   ```bash
   kubeseal --cert ../cert.pem \
     --format yaml \
     --scope namespace-wide \
     < gelift-tracker-secrets.k3s.yaml \
     > ../overlays/k3s/sealed-gelift-tracker-secrets.yaml
   ```

4. Delete the plaintext file:

   ```bash
   rm gelift-tracker-secrets.k3s.yaml
   ```

5. Commit `overlays/k3s/sealed-gelift-tracker-secrets.yaml`. It is safe to publish; only the in-cluster Sealed Secrets controller can decrypt it.

## Rotating secrets

Repeat steps 2–4. The controller will pick up the updated `SealedSecret` and refresh the derived `Secret`.

## Optional keys

- **`DATABASE_URL`** — Add under `stringData` when using PostgreSQL instead of the SQLite file on the PVC (omit or leave unset for the default SQLite deployment).

## Initial checkout

The committed `sealed-gelift-tracker-secrets.yaml` decrypts to placeholder MQTT values (`REPLACE_ME`). Seal real credentials before relying on MQTT in production.
