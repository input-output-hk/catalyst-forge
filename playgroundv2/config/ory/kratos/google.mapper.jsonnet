// Domains allowed to register (Google Workspace hosted domains)
local allowed_domains = [
  'iohk.io',
];

// ---- OIDC claims from Kratos (don't change) ----
local claims = { email_verified: false } + std.extVar('claims');

// ---- Normalize and validate hd ----
local has_hd = 'hd' in claims && claims.hd != null && claims.hd != '';
local hd_lc = if has_hd then std.asciiLower(claims.hd) else null;
local allowed_domains_lc = std.map(function(d) std.asciiLower(d), allowed_domains);

if !has_hd then
  error "This Google account is not part of a Workspace organization (no 'hd' claim)."
else if std.member(allowed_domains_lc, hd_lc) == false then
  error 'Your Google Workspace domain is not allowed to register.'
else if !(claims.email_verified) then
  error 'Your Google email is not verified. Please verify it in Google and try again.'
else if !('email' in claims) || claims.email == null || claims.email == '' then
  error 'Missing email claim from Google.'
else
  {
    identity: {
      traits: {
        // Map only when all checks pass
        email: claims.email,
        domain: hd_lc,  // store the Workspace domain if your schema has this trait
        // first_name: claims.given_name,
        // last_name:  claims.family_name,
      },
    },
    metadata_public: {
      oidc_provider: 'google',
      oidc_subject: claims.sub,
      hd: claims.hd,
    },
  }
