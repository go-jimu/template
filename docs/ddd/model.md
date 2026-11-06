# Domain Model

This file is the authority for business meaning in the template example.
It describes only the accepted example model. Runtime, package, database, and
broker choices belong in [design.md](design.md).

## Ubiquitous language

| Term | Meaning |
| --- | --- |
| User | A person identity with profile and credential data owned by the User context. |
| Registration | Creation of a valid User with a new stable identity. |
| Credential | The secret used to prove control of a User identity. |
| User Created | The internal User-context fact recorded after registration. |
| User Registered | The stable fact published for other contexts after a User was created. |
| Welcome Notification | The Notification context's reaction to a registered User. |

`UserCreated` and `UserRegisteredV1` are not interchangeable. The first uses
the User context's internal language; the second is published language for
external consumers.

## User bounded context

### Authority

The User context is the source of truth for User identity, display name, email,
and credentials. `User` is the consistency and lifecycle owner for these
facts.

### Registration

A registration request supplies a name, email, and initial password. A
successful registration:

1. trims the display name;
2. trims and lowercases the email;
3. accepts a password containing between 8 and 72 bytes;
4. creates a new stable User identity; and
5. records the `UserCreated` fact.

Registration is rejected when the resulting User data is invalid. The current
example does not define email or display-name uniqueness, account activation,
or a deletion lifecycle; do not infer those policies from common account
systems or from database columns.

### Password change

A password change targets one existing User. The current password must match
the stored credential and the replacement password must contain between 8 and
72 bytes. A rejected change leaves the credential unchanged.

### Invariants

- A User always has a stable identity.
- Display name is required and contains at most 24 characters.
- Email is required, normalized to lowercase, syntactically valid, and contains
  at most 48 characters.
- A stored credential is required; raw passwords are never part of the
  persisted User state.
- Credential changes must pass through User behavior and verify the current
  password first.

### Failure tolerance

A User is registered together with the commitment to announce it: the
registration stores `UserRegisteredV1`'s pending publication with the User,
or neither. Publication then retries until it succeeds, at least once; a
publication that can never succeed, such as an undecodable event, stops and
waits for a restart or an operator. No publication or welcome-notification
failure compensates or rolls back registration.

## Notification bounded context

The Notification context owns the reaction named `SendWelcomeNotification`.
The example intentionally does not model notification delivery state,
provider selection, retries, or a Notification Aggregate because it owns no
business lifecycle or invariant yet.

`UserRegisteredV1` is treated as a historical snapshot of the registered User.
The Notification context translates that published language into its own: the
registered User becomes a `Recipient`, and the message ID is the `Reference`
back to the triggering fact. One registration keeps one message ID across
every redelivery, so `Reference` identifies it. Repeated delivery may repeat
the observable example behavior; a real sender deduplicates by `Reference`.

## Context relationship

The User context is upstream and publishes `UserRegisteredV1` as stable
published language. Notification is downstream and consumes that contract
without importing User Domain or Application types.

Cross-context collaboration uses Integration Messages. Domain Events never
cross the User context boundary, and contexts do not share Repositories,
database tables, Aggregate instances, or internal Application services.
