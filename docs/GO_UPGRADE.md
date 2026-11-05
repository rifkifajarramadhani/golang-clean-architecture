# Go 1.27.3 compatibility review

Reviewed on November 5, 2026. This project now matches typefn's backend at Go
1.27.3, replacing Go 1.26.5 in the module, Docker builder, CI, and contributor
documentation. CI uses golangci-lint v2.14.0 and govulncheck v1.8.0.

## Confirmed regression and fix

Go 1.27 updates its Unicode data from version 15 to 17, changing lowercase
mappings. See the [Go 1.27 release notes](https://go.dev/doc/go1.27#unicode).
Comparison with the locally installed Go 1.26.5 toolchain found 55 changed
mappings. Both toolchains accept the example address below.

Previously, `NormalizeEmail("\u1C89@example.com")` retained U+1C89. Go 1.27's
`strings.ToLower` changes it to U+1C8A. MySQL 8.4's default collation treats those
values as different keys, so login and verification lookups could miss an
existing User. Some new mappings could also merge previously distinct keys.

Email normalization now lowercases characters assigned in Unicode 15.0.0 and
retains characters unassigned in that version. It uses assignment data from the
existing `golang.org/x/text` v0.39.0 dependency. Its version is unchanged; it is
now a direct dependency. Bootstrap Admin configuration uses the same domain
normalizer to avoid changing the key before authentication receives it.

Regression tests cover all 55 mappings, existing casing behavior, Bootstrap
Admin configuration, and MySQL lookups of historical active and Pending Emails.
They failed before the fix and pass afterward. No data migration or public
interface change is required for this fix.

## Other compatibility checks

| Area | Result |
| --- | --- |
| JSON | Go 1.27 preserves the v1 API's marshaling behavior, although error wording can change. HTTP decoding returns a fixed application error. JWT and queue tests pass with existing JSON imports. |
| Timers | The removed `asynctimerchan` override is unused. Scheduler and worker timers use cancellation and channel selection without depending on buffered timer channels. Race tests pass. |
| SMTP/TLS | The transport already requires TLS 1.2. No removed TLS compatibility overrides are configured in the repository. External SMTP handshakes were not exercised. |
| Template archives | Compression output may differ between Go releases. Repeated builds with Go 1.27.3 remain identical and their generated checksums verify. Existing published checksums should remain attached to their original archives. |
| Platforms | Docker and native checks ran on Linux amd64. Native macOS development now requires macOS 13 or later; macOS was not tested. |

These checks follow the [Go 1.27 release notes](https://go.dev/doc/go1.27).
No further application code incompatibilities were found in the reviewed paths.

## Validation and remaining finding

Validation includes `make check` with both MySQL integration DSNs set, covering
formatting, vet, lint, unit tests, integration tests, and race detection. Workflow
validation and `go mod tidy -diff` pass. Production and development Docker targets
build with Go 1.27.3. Template packaging checks cover version pins, repeated
archive bytes, and SHA-256 checksums.

`make vuln` still fails on the pre-existing
[GO-2026-6629 advisory](https://pkg.go.dev/vuln/GO-2026-6629). The scanner reports a
reachable panic in `golang.org/x/text/secure/precis`, through SMTP delivery from
`Transport.Send`. The project pins x/text v0.39.0; the fix is in v0.41.0.
The same finding was reproduced with the original module file. Dependency
security updates remain separate from this toolchain upgrade.
