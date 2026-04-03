RequireFamily now only handles family membership.

Changed RequireFamily to use a defensive nil-auth guard that returns InternalServerError("RequireFamily: authentication middleware not applied", nil), and updated the comment to require chaining after RequireAuth.

Renamed the nil-auth test to TestRequireFamily_NilAuth_DefensiveGuard and updated its logs to describe the misconfiguration case.

Verification passed:
- lsp_diagnostics on internal/middleware/auth.go and auth_test.go: no diagnostics found
- go build ./...: PASS
- go test ./internal/middleware/... -v: PASS
