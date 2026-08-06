# ADR 0002: Registration and Verification Delivery Boundaries

- Status: Accepted
- Date: 2026-08-06

## Context

Registration creates a User and a single-use email-verification token, then
dispatches the verification message. Verification consumes that token, updates
identity state, and may dispatch a welcome message after a First Verification.
Database transactions cannot atomically commit arbitrary external queue or mail
effects.

## Decision

Registration invokes verification dispatch inside the User-creation
transaction. If dispatch fails, the transaction returns an error and rolls back
the User and token. This prevents a committed User from being left without an
attempted verification delivery.

The accepted trade-off is asymmetric: an external dispatch may succeed and the
database commit may subsequently fail. That already-dispatched effect cannot be
rolled back atomically. Dispatch implementations must remain bounded by the
request context because the transaction stays open during dispatch.

Email verification commits the identity transition before welcome delivery. A
welcome message is sent only after First Verification. Welcome dispatch is
best-effort: its failure does not revert or report failure for the completed
verification. Pending Email verification does not send another welcome message.

## Consequences

- Registration reports dispatch failure and leaves no committed User from that
  attempt.
- A rare dispatch-without-commit outcome remains possible for external queues.
- Verification remains successful even when welcome delivery is unavailable.
- Delivery handlers and messages must tolerate retries and duplicates.
- A transactional outbox is the preferred future improvement when stronger
  atomicity between identity state and dispatch is required.
