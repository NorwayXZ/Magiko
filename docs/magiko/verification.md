# Verification — 0.1.0-preview.1

Host: macOS ARM64, Go 1.26.8, Node 26.12.0, Playwright 1.64.1.
All application tests use isolated HOME/configuration and synthetic credentials.
The desktop app was not installed into Applications or pointed at live Codex.

## Checks

| Check | Result |
| --- | --- |
| `go test -tags nogui ./...` | Passed (88 package entries; 66 tested, 22 without tests) |
| Fork integration tests, five packages | Passed |
| Native subscription HTTP → fixture upstream → SSE, headers/context preserved | Passed |
| `go vet -tags 'nogui magiko' ./...` | Passed on host |
| Linux/Windows AMD64 `CGO_ENABLED=0 go vet -tags 'nogui magiko' ./...` | Both passed |
| Workspace browser contracts (Chromium + WebKit, five languages) | 12 tests passed |
| Existing plugin-kernel/builtins/i18n browser regressions | 11 tests passed |
| `gofmt -l` and `git diff --check` | Clean |
| Embedded Python helper syntax | Five files parsed without executing |
| `build/magiko/build.sh` | Native ARM64 app and ZIP built |
| Info.plist, `codesign --verify --deep --strict` | Passed, ad-hoc signature |
| Native executable in isolated HOME | `Magiko 0.1.0-preview.1` |
| `python3 build/magiko/smoke.py` | Embedded UI/preferences API/web authentication passed |
| ZIP checksum | SHA-256 verified |

The first full suite identified the core script budget exceeded by 67 bytes.
This was fixed by sharing the existing dirty-page navigation guard instead of
adding another duplicated branch; the budget was not increased. Initial
built-in-page browser checks also exposed a fork tab appearing in non-fork
fixtures; the workspace now requires its explicit boot flag and creates its
own navigation entry. Both regressions have direct follow-up checks.

The build still emits the upstream tray-image enum-conversion and duplicate
`-lobjc` warnings. The Mach-O minimum deployment target is 12.0; older macOS
versions have not been runtime-tested. The distributable is not Developer-ID
signed or notarized.

## Reproduce

```sh
go test -tags nogui ./...
go test -tags 'nogui magiko' ./internal/magiko ./internal/brand ./internal/gui ./internal/gateway ./internal/update \
  -run 'TestMagiko|TestPreferences|TestPreferenceStorage|TestSkillPack|TestCoreScriptBudget' -count=1
go vet -tags 'nogui magiko' ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go vet -tags 'nogui magiko' ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go vet -tags 'nogui magiko' ./...
npm install --prefix /tmp/magiko-ui-deps --no-audit --no-fund playwright@1.64.1
/tmp/magiko-ui-deps/node_modules/.bin/playwright install chromium webkit
NODE_PATH=/tmp/magiko-ui-deps/node_modules node --test internal/gui/tests/magiko.test.cjs
NODE_PATH=/tmp/magiko-ui-deps/node_modules node --test --test-concurrency=1 \
  internal/gui/tests/plugin-kernel.test.cjs internal/gui/tests/builtins.test.cjs internal/gui/tests/i18n-keys.test.cjs
build/magiko/build.sh
python3 build/magiko/smoke.py
```

## Not verified

- Real ChatGPT subscription login, refresh, quota and billing: no real account
  was used. The native subscription path uses an `httptest` upstream with
  synthetic account headers and streaming replies.
- Native window/tray interaction, macOS Login Items and daemon installation:
  code builds, but no live installation/startup changes were made.
- Entire inherited UI suite and every third-party provider: only the listed
  regressions ran. Non-macOS distributables are not published.
- So-called jailbreak effects: not implemented or claimed. Request preferences
  are ordinary context and do not change the upstream model's safeguards.

The repository screenshot shows isolated fixture data, not a live account.
