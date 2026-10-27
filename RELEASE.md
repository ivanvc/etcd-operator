# Release Process

The etcd-operator is released on an as-needed basis. Releases are built and
published by [GoReleaser](https://goreleaser.com) from the
[Release workflow](.github/workflows/release.yaml), which runs when a version
tag is pushed. Each release produces:

* A multi-architecture (`linux/amd64`, `linux/arm64`) container image,
  `gcr.io/etcd-development/etcd-operator:$VERSION`.
* A GitHub Release with the changelog, the `install.yaml` installation file, and
  its checksum.

While the etcd-operator is pre `v1.0.0`, there are no release branches. Releases
are tagged from the `main` branch.

## Releasing a new version

1. An issue is opened proposing a new release, with the highlights since the
   last release.
1. We get [OWNERS](OWNERS) concensus for this release (LGTM).
1. An OWNER picks a commit from `main` with passing postsubmit jobs, and creates
   and pushes a signed tag (where `upstream` is the remote for
   `git@github.com:etcd-io/etcd-operator.git`):

   ```bash
   git tag --sign "${VERSION}" --message "${VERSION}" "${COMMIT}"
   git push upstream "${VERSION}"
   ```

1. The Release workflow builds and pushes the container image, and creates a
   draft GitHub Release. Pre-release versions (e.g., `v0.3.0-rc.0`) are marked as
   pre-releases.
1. The OWNER reviews and edits the draft release notes, and publishes the
   release.
1. The release issue is closed.
1. The release is announced on the mailing lists.

## Testing the release configuration

Pull requests changing the release configuration run a GoReleaser snapshot
build in the [Verify Release workflow](.github/workflows/release-verify.yaml).
To run it locally (no images are pushed, and nothing is published):

```bash
goreleaser release --snapshot --clean
```

The artifacts are written to the `_output` directory.

## Prerequisites

The Release workflow pushes images to `gcr.io/etcd-development` using
[Workload Identity Federation through a Service Account](https://github.com/google-github-actions/auth#indirect-wif).
It requires the following repository variables:

* `GCP_WORKLOAD_IDENTITY_PROVIDER`: The full identifier of the workload identity
  provider, in the form
  `projects/<number>/locations/global/workloadIdentityPools/<pool>/providers/<provider>`.
* `GCP_SERVICE_ACCOUNT`: The email of the service account used to push the
  images.

These values are identifiers, not credentials, so they don't need to be
secrets. Access is restricted by the following configuration, which must be in
place:

1. The workload identity provider only accepts tokens from the Release
   workflow's job, which runs in the `release` environment. Its attribute
   condition must match the token's subject exactly:

   ```
   assertion.sub == 'repo:etcd-io/etcd-operator:environment:release'
   ```

   Broader conditions, such as `assertion.repository_owner == 'etcd-io'`, would
   allow any workflow from any repository in the organization to push images.
1. The `roles/iam.workloadIdentityUser` role on the service account is only
   granted to that subject, not to the whole pool:

   ```
   principal://iam.googleapis.com/projects/<number>/locations/global/workloadIdentityPools/<pool>/subject/repo:etcd-io/etcd-operator:environment:release
   ```

   The service account only has permissions to push to
   `gcr.io/etcd-development/etcd-operator`.
1. The `release` environment in the GitHub repository settings
   (Settings → Environments) only allows deployments from tags matching `v*`,
   so workflows running from branches can't use it. Optionally, it requires a
   reviewer's approval before the Release workflow runs.
