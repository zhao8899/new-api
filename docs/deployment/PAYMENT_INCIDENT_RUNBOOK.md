# Payment Incident Runbook

This runbook covers refund, missing topup, manual fulfillment, and payment
dispute handling for Alipay and WeChat Pay through the Epay gateway.

## Roles

- Billing owner: approves refunds, manual fulfillment, and dispute responses.
- Operator: checks gateway orders, exports reconciliation reports, and executes manual fulfillment.
- Support: communicates with the customer and collects evidence.

Never ask customers for passwords, API keys, or full payment credentials.

## Missing Topup

Use this flow when the customer paid but quota was not credited.

1. Collect `trade_no`, user id, payment method, payment time, and customer receipt screenshot.
2. In the gateway dashboard, verify the provider order status is successful.
3. In new-api admin billing history, search the `trade_no`.
4. Export admin reconciliation CSV for the payment window.
5. If the order is pending and the gateway order is successful, use admin manual completion.
6. If there is no new-api order, do not create quota blindly. Escalate to billing owner and attach gateway proof.
7. Record the action in the incident ticket with operator, time, evidence, and quota credited.

Manual completion is only allowed after gateway success is verified.

## Duplicate Callback Or Duplicate Topup

The Epay callback handler is idempotent, so duplicate successful callbacks
should not double-credit quota. If a duplicate credit is suspected:

1. Search topup history by `trade_no`.
2. Check user quota change logs for repeated topup entries.
3. Compare gateway order count and new-api order count in reconciliation.
4. If duplicate credit is confirmed, freeze further manual actions and escalate to billing owner.

## Refund

Refunds are handled in the payment gateway, then reflected operationally in
new-api records and support notes.

1. Verify the original topup order, user id, amount, and gateway transaction id.
2. Confirm whether the quota was consumed.
3. Billing owner approves or rejects the refund.
4. Execute refund in the gateway dashboard.
5. Export gateway refund evidence and attach it to the ticket.
6. If quota clawback is required, handle it with an explicit admin operation and log the reason.
7. Notify the customer with refund amount, payment method, and expected arrival time.

## Dispute Or Chargeback

1. Preserve order, callback logs, user id, IP, receipt, and quota usage evidence.
2. Disable manual fulfillment on the order until the dispute is resolved.
3. Respond through the gateway dispute workflow with evidence.
4. If the dispute is lost, reconcile the loss in the billing report.
5. If fraud is suspected, suspend the user after billing owner approval.

## Reconciliation Procedure

Run daily:

```powershell
# From project root after logging into the admin UI in the same browser session:
# Admin UI path: System Settings -> Billing -> Payment Reconciliation -> Export CSV
```

Compare:

- Gateway settlement export by payment method.
- new-api reconciliation CSV grouped by provider, method, and status.
- Pending order list older than 10 minutes.

Differences must have one of these final dispositions:

- Gateway success, new-api pending: manual completion after verification.
- Gateway failed or unpaid, new-api pending: leave pending or expire according to policy.
- Gateway refund, new-api success: refund ticket and quota clawback decision.
- Unknown: escalate to billing owner.

## Evidence Template

```text
Incident:
Customer:
User ID:
new-api trade_no:
Gateway transaction id:
Payment method:
Gateway status:
new-api status:
Amount:
Quota:
Evidence links:
Operator:
Decision:
Action taken:
Completed at:
```
