# Domestic Payment Operations

This project uses the existing Epay-compatible gateway path for mainland China wallet top-ups. Configure Alipay and WeChat Pay through an Epay-compatible aggregator, then verify callback signing and idempotent settlement before commercial launch.

## Required Configuration

Configure these values in the admin payment settings:

| Setting | Purpose |
| --- | --- |
| `PayAddress` | Epay-compatible gateway base URL, for example `https://pay.example.com`. |
| `EpayId` | Merchant ID from the payment gateway. |
| `EpayKey` | Gateway signing key. Treat as a production secret. |
| `CustomCallbackAddress` | Public HTTPS origin used for callbacks when `ServerAddress` is not externally reachable. |
| `PayMethods` | JSON array containing `alipay` and `wxpay` methods. |
| `Price` | Local currency price per quota unit displayed by the wallet page. |
| `MinTopUp` | Minimum top-up amount. |

Recommended `PayMethods`:

```json
[
  {
    "name": "支付宝",
    "color": "#1677FF",
    "type": "alipay"
  },
  {
    "name": "微信支付",
    "color": "#07C160",
    "type": "wxpay"
  }
]
```

## Callback URLs

Wallet top-up callback:

```text
https://your-domain.example/api/user/epay/notify
```

Subscription callback:

```text
https://your-domain.example/api/subscription/epay/notify
```

Both endpoints accept Epay-compatible signed callbacks. The wallet callback completes the order and credits user quota inside one database transaction; duplicate callbacks are idempotent and do not credit twice.

## Launch Checklist

- [ ] Payment compliance confirmation has been completed in the admin payment settings.
- [ ] Public callback address is HTTPS and reachable from the payment gateway.
- [ ] `PayAddress`, `EpayId`, and `EpayKey` are configured and not logged.
- [ ] `PayMethods` contains `alipay` and `wxpay`.
- [ ] Test Alipay top-up creates a pending order, receives a signed callback, marks the order success, and credits quota once.
- [ ] Test WeChat Pay top-up creates a pending order, receives a signed callback, marks the order success, and credits quota once.
- [ ] Duplicate callback for the same order does not change user quota a second time.
- [ ] Invalid callback signature returns `fail` and does not credit quota.
- [ ] Admin manual completion requires secure verification and is covered by an operator runbook.
- [ ] Daily reconciliation compares payment gateway successful trades with `topups` rows and user quota deltas.
- [ ] Refund handling is documented as an operator workflow before enabling refunds to users.

## Verification Commands

```bash
go test ./controller ./model
go test ./...
```

For Docker smoke tests, verify both the API and frontend:

```bash
curl -fsS http://localhost:3000/api/status
curl -fsS http://localhost:3000/api/user/topup/info
```

The top-up info endpoint requires user authentication in normal deployments.

## Reconciliation Data

Use these fields for reconciliation:

| Table | Fields |
| --- | --- |
| `topups` | `trade_no`, `user_id`, `amount`, `money`, `payment_method`, `payment_provider`, `status`, `create_time`, `complete_time` |
| `logs` | top-up log rows with payment method metadata in `other` |
| `users` | `quota` balance after successful settlement |

Every successful gateway trade should have exactly one successful `topups` row and one corresponding quota increase.

## Internal Reconciliation API

Administrators can query grouped top-up totals for daily payment checks:

```text
GET /api/user/topup/reconciliation?start_time=1780400000&end_time=1780486400&payment_provider=epay
```

Optional filters:

| Query | Meaning |
| --- | --- |
| `start_time` | Unix timestamp lower bound on `create_time`. Defaults to the last 24 hours. |
| `end_time` | Unix timestamp upper bound on `create_time`. Defaults to now. |
| `payment_provider` | Gateway provider, for example `epay`. |
| `payment_method` | Payment method, for example `alipay` or `wxpay`. |
| `status` | Internal order status, for example `success`, `pending`, or `expired`. |

Use this API to compare grouped internal totals with the payment gateway's daily successful trade export. Investigate any mismatch before closing the daily账务 batch.
