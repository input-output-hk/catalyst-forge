# Catalyst Forge v2 Architecture

WIP architecture notes for Forge v2.

## Dump

- [ ] Stand up initial API endpoint for receiving new build from GHA
  - [ ] Identify initial incoming payload from GHA
  - [ ] Implement validation layer to only allow trusted repositories to submit builds


## Build Submission

Execution begins when a GHA workflow executes an API call to Catalyst Forge API.
Workflow uses Catalyst Forge CLI to execute the request.
The CLI collects all required runtime data, including a valid GitHub OIDC token, and POSTSs to API build endpoint.
The API server performs initial validation of the build request:
  1. Validates incoming GHA JWT via public GHA JWKS
    - Returns 401 and audit entry if validation fails
  1. Uses the source repository name (sourced from JWT claims) to validate if repository is approved to submit build.
    - Returns 401 and audit entry if repository is not found
  1. Uses GH API to validate source commits exists on source repository
    -  Returns 400 if not found
Upon validation success, API creates new build entry in backend and hands off build to discovery agent.