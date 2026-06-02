# Header Forwarding Optimization Record

## Summary

This record tracks the hardening change that prevents ingress/client identity headers from being forwarded to upstream providers.

- Repository: `new-api`
- Commit baseline inspected before change: `fbf235d2`
- Change area: unified upstream request header handling
- Status: implemented

## Background

`new-api` can receive requests through multiple ingress layers such as reverse proxies, Cloudflare Tunnel, and Cloudflare Access. Those ingress layers may inject or preserve client identity headers like:

- `X-Forwarded-For`
- `Forwarded`
- `CF-Connecting-IP`
- `True-Client-IP`
- `X-Real-IP`
- `CF-Access-*`

The gateway should not leak those ingress-side headers to upstream AI providers. Upstream providers should normally only see:

- the gateway egress IP
- provider-required authentication headers
- provider-required protocol headers

## Problem

Before this change, the project supported several dynamic header forwarding paths:

- wildcard / regex header passthrough in `HeaderOverride`
- `{client_header:...}` placeholders
- runtime header overrides applied before the upstream request is sent

Those features were flexible, but they also made it possible to accidentally forward ingress/client identity headers to upstream providers.

## Goal

Enforce a single rule at the unified upstream request layer:

- ingress/client identity headers are available for local gateway logic if needed
- ingress/client identity headers are never sent to upstream providers

## Implementation

The protection was implemented in:

- [`relay/channel/api_request.go`](../../relay/channel/api_request.go)

The change blocks sensitive headers in three places:

1. passthrough rules
2. `{client_header:...}` placeholder resolution
3. final application of header overrides to the upstream request

## Blocked Headers

The current blocked header set is:

- `Forwarded`
- `X-Forwarded-For`
- `X-Forwarded-Host`
- `X-Forwarded-Port`
- `X-Forwarded-Proto`
- `X-Forwarded-Protocol`
- `X-Real-IP`
- `CF-Connecting-IP`
- `True-Client-IP`
- `CF-Access-*`

## Files Changed

- [`relay/channel/api_request.go`](../../relay/channel/api_request.go)
- [`relay/channel/api_request_test.go`](../../relay/channel/api_request_test.go)

## Test Coverage Added

Added tests verify:

- wildcard passthrough does not leak sensitive ingress headers
- `client_header` placeholders cannot expose blocked headers upstream
- runtime header overrides cannot force blocked headers into upstream requests
- ordinary non-sensitive headers still pass through as expected

## Verification

Executed:

```powershell
go test ./relay/channel -run "Test(ProcessHeaderOverride|ApplyHeaderOverrideToRequest)" -count=1
```

Result:

- passed

## Compatibility Notes

This change should not affect standard upstream AI providers because these blocked headers are not required by normal provider APIs.

Potential behavior change only applies if a custom upstream explicitly depended on ingress/client identity headers. In that case, the custom upstream will now only observe the gateway's own egress identity.

## Follow-up Recommendations

- Review channel configs for broad `HeaderOverride` passthrough rules such as `*` or permissive regexes.
- Avoid using `pass_headers`, `copy_header`, or `{client_header:...}` for ingress-origin metadata unless there is a clear local-only need.
- Keep provider-specific required headers inside the relevant adaptor instead of relying on generic passthrough.
