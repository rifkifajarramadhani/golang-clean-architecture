# Project Context

This glossary defines domain language shared by the template source and every
project generated from it. It describes concepts, not implementation choices.

## Project Language

- **Template Source:** This versioned repository in its role as the source
  material from which projects can be generated.
- **Runnable Default:** The complete, executable configuration kept in the
  Template Source to demonstrate one supported combination without generation.
- **Generated Project:** A standalone project created from the Template Source.
  It is owned and evolved independently after generation.
- **Selected Stack:** The set of framework, database, queue, scheduler, and
  related options chosen for a Generated Project. The Runnable Default is one
  Selected Stack, not a requirement imposed on every Generated Project.

## Identity Language

- **User:** The canonical identity that can authenticate and use the system.
- **Role:** A named set of privileges assigned to a User.
- **Standard User:** A User with the ordinary, non-administrative Role.
- **Admin:** A User whose Role permits administrative operations.
- **Verified Email:** The User's active email address after successful proof of
  ownership.
- **Pending Email:** A proposed replacement email address awaiting proof of
  ownership. It does not replace the Verified Email until verification succeeds.
- **First Verification:** The transition from an unverified User to a User with
  a Verified Email. Verifying a Pending Email for an already-verified User is
  not a First Verification.
- **Bootstrap Admin:** The first eligible User promoted to Admin through the
  initial administrative bootstrap policy.

Use **User** consistently in code, APIs, documentation, and product language.
Avoid **Account** and **Member** as synonyms because they create ambiguity about
whether they represent authentication, billing, tenancy, or organization
membership.
