package httpkit

import (
    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "net/http"
)

// ErrorCode enumerates auth-domain specific error codes.
type ErrorCode string

const (
    ErrorTokenExpired      ErrorCode = "token_expired"
    ErrorTokenInvalid      ErrorCode = "token_invalid"
    ErrorSessionExpired    ErrorCode = "session_expired"
    ErrorStepUpRequired    ErrorCode = "step_up_required"
    ErrorInsufficientRoles ErrorCode = "insufficient_roles"
    ErrorWebAuthnFailed    ErrorCode = "webauthn_failed"
    ErrorInvalidCredential ErrorCode = "invalid_credential"
    ErrorCredentialExists  ErrorCode = "credential_exists"
    ErrorInviteExpired     ErrorCode = "invite_expired"
    ErrorInviteInvalid     ErrorCode = "invite_invalid"
    ErrorInviteRedeemed    ErrorCode = "invite_redeemed"
    ErrorInviteLocked      ErrorCode = "invite_locked"
    ErrorRecoveryInvalid   ErrorCode = "recovery_invalid"
    ErrorRecoveryExpired   ErrorCode = "recovery_expired"
    ErrorCodeInvalid       ErrorCode = "code_invalid"
)

// Thin constructors returning base httpkit errors with auth-domain codes.
func NewStepUpRequiredError(message string) *basehttpkit.HTTPError {
    if message == "" { message = "Please confirm with your passkey to continue" }
    return &basehttpkit.HTTPError{Status: http.StatusPreconditionRequired, Code: basehttpkit.ErrorCode(ErrorStepUpRequired), Message: message}
}

func NewWebAuthnFailedError(message string) *basehttpkit.HTTPError {
    if message == "" { message = "WebAuthn verification failed" }
    return &basehttpkit.HTTPError{Status: http.StatusBadRequest, Code: basehttpkit.ErrorCode(ErrorWebAuthnFailed), Message: message}
}

func NewInvalidCredentialError(message string) *basehttpkit.HTTPError {
    if message == "" { message = "Invalid credential" }
    return &basehttpkit.HTTPError{Status: http.StatusBadRequest, Code: basehttpkit.ErrorCode(ErrorInvalidCredential), Message: message}
}

func NewCredentialExistsError(message string) *basehttpkit.HTTPError {
    if message == "" { message = "Credential already exists" }
    return &basehttpkit.HTTPError{Status: http.StatusConflict, Code: basehttpkit.ErrorCode(ErrorCredentialExists), Message: message}
}

func NewInviteExpiredError() *basehttpkit.HTTPError {
    return &basehttpkit.HTTPError{Status: http.StatusBadRequest, Code: basehttpkit.ErrorCode(ErrorInviteExpired), Message: "Invite expired"}
}

func NewInviteInvalidError() *basehttpkit.HTTPError {
    return &basehttpkit.HTTPError{Status: http.StatusBadRequest, Code: basehttpkit.ErrorCode(ErrorInviteInvalid), Message: "Invalid invite"}
}

func NewInviteRedeemedError() *basehttpkit.HTTPError {
    return &basehttpkit.HTTPError{Status: http.StatusBadRequest, Code: basehttpkit.ErrorCode(ErrorInviteRedeemed), Message: "Invite already redeemed"}
}

func NewInviteLockedError() *basehttpkit.HTTPError {
    return &basehttpkit.HTTPError{Status: http.StatusTooManyRequests, Code: basehttpkit.ErrorCode(ErrorInviteLocked), Message: "Invite locked due to attempts"}
}

func NewRecoveryInvalidError() *basehttpkit.HTTPError {
    return &basehttpkit.HTTPError{Status: http.StatusBadRequest, Code: basehttpkit.ErrorCode(ErrorRecoveryInvalid), Message: "Invalid recovery attempt"}
}

func NewRecoveryExpiredError() *basehttpkit.HTTPError {
    return &basehttpkit.HTTPError{Status: http.StatusBadRequest, Code: basehttpkit.ErrorCode(ErrorRecoveryExpired), Message: "Recovery code expired"}
}
