# Standalone results relay (Terraform)

**Not an approved public deployment. Authentication is deferred.** All three
routes deliberately retain `NONE` authorization. Anyone able to reach the API
can submit fabricated results, receive private match data and receipt handles,
and acknowledge/delete deliveries. CORS and throttles are NOT authentication.
Resolve the exposure/access-control release gate with the owner before rollout;
this conversion does not introduce an identity service.

This is the supported standalone deployment root. The obsolete SAM template and
saved deployment configuration have been removed; their history remains in Git.
Removing these files does not delete or change any existing AWS stack or queue.
Reader source and receive/commit/ack semantics are unchanged.
See [Stage 02](../../../docs/rollout/stage-02.md) for the delivery contract.

## Build and offline verification

Prerequisites: Node **22**, npm, Info-ZIP `zip`, Terraform >=1.7 and <2.0.
Run from the repository root with Node 22 first on `PATH`:

```sh
node score-scrape/terraform/build-reader.mjs
npm --prefix score-scrape/reader ci --omit=dev --ignore-scripts --no-audit --no-fund
node --test score-scrape/reader/*.test.js
node --test score-scrape/terraform/safety.test.mjs
terraform -chdir=score-scrape/terraform init -backend=false
terraform -chdir=score-scrape/terraform fmt -check -recursive
terraform -chdir=score-scrape/terraform validate
terraform -chdir=score-scrape/terraform test
```

Tests use fully mocked AWS plans plus local static assertions for computed
resource bindings and queue lifecycle protection. No apply, credentials or AWS calls.
Provider downloads use the official registry. Commit both dependency lockfiles;
never commit build output, node_modules, state, plans or private tfvars.
The build copies only `index.js`, package manifests and locked production SDK
dependencies into a fresh ignored directory. `npm ci --ignore-scripts` avoids
install hooks; timestamps, file permissions and ZIP order are normalized. Use the
same Node/npm/zip toolchain for byte-identical rebuilds. No runtime-installed SDK
is assumed. Build again after source/lock changes, **before each plan**; Terraform
hashes the resulting ZIP for Lambda updates. There are no build/deploy provisioners.

## Operator-only deployment (not performed by this change)

If this configuration is already deployed, preserve its existing Terraform state,
variables, account, region and name prefix when changing branches/checkouts. Do not
initialize a second local state for the same resources. This cleanup changes no
Terraform resource addresses and requires no AWS replacement. Review the plan;
stop if it proposes replacing a queue or recreating an existing deployment.

Default: create a **NEW standalone API URL, source queue and DLQ** in an explicitly
chosen account/profile, region and unique name prefix. No existing-account read
access was available. This is not an in-place migration and does not adopt the old
CloudFormation stack. Confirm the chosen account with your normal authorized
operator process; Terraform intentionally does not hardcode a profile/account.

After the security and Stage 02 disposable-Postgres durability gates are satisfied:

```sh
# Choose your own authorized profile and region; do not reuse legacy resource names.
export AWS_PROFILE='<your-authorized-profile>'
export TF_VAR_region='<target-region>'
export TF_VAR_name_prefix='<unique-new-relay-prefix>'
node score-scrape/terraform/build-reader.mjs
terraform -chdir=score-scrape/terraform init
terraform -chdir=score-scrape/terraform plan -out=relay.tfplan
# Review account, new resources, IAM, artifact hash, costs and all changes first.
terraform -chdir=score-scrape/terraform apply relay.tfplan
terraform -chdir=score-scrape/terraform output
```

These are handoff commands, not authorization for an agent to run them. The initial
plan must only create the intended standalone resources, never replace old queues.
The root defaults to local state: keep it private, encrypted/backed up and under
single-operator control; agree a locked remote backend before shared operation.
Local state and saved plans can contain sensitive values. Do not lose the state.
Optional variables: `allowed_origin` (one exact HTTPS origin, default
`https://play.autodarts.com`), `alarm_action_arns` (default empty),
`log_retention_days` (default 30; explicit finite retention for the new log group).
Configure and verify operator alert routing before calling this monitored; empty
actions create metric alarms but send no notifications.

## Ownership and backlog

- Existing messages **do not move automatically**. Coordinate with the old stack
  owner to preserve/drain/export the backlog and DLQ before switching. An operator
  without old-account access cannot verify or migrate that backlog. Track retention
  deadlines and agree a maintenance/cutover window; pause producers if necessary.
- Source retention is four days; DLQ retention is 14 days. Standard SQS messages
  retain their original enqueue age when moved to a DLQ. These are bounded buffers,
  not an archive. No automatic archive, replay or migration is provided.
- Both queues have `prevent_destroy`. Terraform rejects planned destruction or
  replacement while that lifecycle rule remains in configuration. Unlike
  CloudFormation `Retain`, it does not detach and preserve a queue while deleting
  a stack, and removing the resource block also removes this protection. It does
  not stop expiry, message deletion, console/API changes or loss of state. Do not
  remove it to get a plan through; never purge queues or delete the old stack as
  a migration shortcut.
- Adoption is a separate owner-led operation, not the default here: require
  explicit owner permission, inventory of exact physical names/configuration,
  CloudFormation retain/detach ownership handoff and a reviewed **no-replacement**
  Terraform plan. Never have CloudFormation and Terraform manage the same resource
  simultaneously. There is no one-command import that guarantees parity.

## Cutover and rollback

1. Agree backlog handling with the old owner; preserve both old queues. Pause ALL
   manual/scheduled polling and separate readers. Clear backend `resultsEndpoint`
   / `RESULTS_ENDPOINT` and restart consumers so polling is actually disabled.
   Never use the old destructive GET reader to drain data.
2. Create/review the new standalone deployment while polling remains paused.
   Arrange producer pause/switch timing so new deliveries cannot be stranded in
   the old queue. Update **both** the browser scraper submission URL to
   `submission_url` and backend reader URL to `read_url`. The backend derives
   `/results/ack`; do not configure it to poll the acknowledgement URL.
3. Verify the backend uses real Postgres, required uniqueness exists, and the
   paired receive/ack reader is deployed. Under controlled polling, submit one
   approved test result, observe a committed **pending import** (not an automatic
   approved score), and then acknowledgement. Check lost-ack replay/idempotency
   in the approved test environment before normal polling resumes.
4. Resume normal consumers only after verification; observe receive/commit/ack
   failures, source age and routed DLQ alarms. Reconcile old backlog explicitly.

Rollback: pause polling and producers first; leave the reader endpoint empty
until a compatible receive/ack endpoint is verified. Preserve BOTH deployments'
queues/DLQs and pending rows, reconcile deliveries and endpoints before resuming.
Do not return to the destructive old reader, purge data, delete either stack or
blindly replay a DLQ. This change does not alter any live app configuration.
