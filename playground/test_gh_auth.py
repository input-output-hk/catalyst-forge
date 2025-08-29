import base64
import hashlib
import json
import time
from typing import Any, Dict

import requests


# --- Configuration ---
# Public endpoints (playground)
MOCK_OIDC_BASE = "https://oidc.projectcatalyst.dev"
HYDRA_PUBLIC_URL = "https://auth.projectcatalyst.dev/hydra/public"

# Hydra client configured for JWT-bearer
CLIENT_ID = "gha-ci"

# Target audience for the GitHub OIDC token must be the Hydra token endpoint URL
HYDRA_TOKEN_URL = f"{HYDRA_PUBLIC_URL}/oauth2/token"
GHA_TOKEN_AUD = HYDRA_TOKEN_URL

# Optional: protected API endpoint to validate the access token
TEST_API_URL = "https://forge.projectcatalyst.dev/api/v1/test"


def jwt_payload(jwt: str) -> Dict[str, Any]:
    parts = jwt.split(".")
    if len(parts) != 3:
        return {}
    pad = "=" * (-len(parts[1]) % 4)
    try:
        return json.loads(base64.urlsafe_b64decode(parts[1] + pad).decode())
    except Exception:
        return {}


def main() -> None:
    # 1) Request a GitHub-like OIDC assertion from the mock OIDC provider
    mint_url = f"{MOCK_OIDC_BASE}/github/actions/token"
    body = {
        "repository": "acme/repo",
        "ref": "refs/heads/main",
        "sha": "deadbeef",
        "actor": "runner",
        "environment": "dev",
        "aud": [GHA_TOKEN_AUD],
    }
    print("Minting mock GitHub OIDC token...")
    r = requests.post(mint_url, json=body, timeout=10, verify=False)
    if r.status_code != 200:
        print("Mint error:", r.status_code, r.text)
        return
    assertion = r.json().get("token")
    if not assertion:
        print("No token returned from mock-oidc")
        return
    print("Assertion (JWT) kid/payload:")
    payload = jwt_payload(assertion)
    print(json.dumps({"payload": payload}, indent=2))

    # 2) Exchange the assertion at Hydra using JWT-bearer grant
    print("\nExchanging assertion at Hydra token endpoint...")
    data = {
        "grant_type": "urn:ietf:params:oauth:grant-type:jwt-bearer",
        "client_id": CLIENT_ID,
        "assertion": assertion,
        "audience": "https://forge.projectcatalyst.dev/api",
        # Optional requested scopes; adjust as needed
        # "scope": "openid",
    }
    tr = requests.post(HYDRA_TOKEN_URL, data=data, timeout=15, verify=False)
    print("Hydra token status:", tr.status_code)
    try:
        token_json = tr.json()
    except Exception:
        print(tr.text)
        return
    print(json.dumps(token_json, indent=2))

    access_token = token_json.get("access_token")
    if not access_token:
        print("No access_token in token response.")
        return

    # Try decoding as JWT (will be empty if opaque)
    at_payload = jwt_payload(access_token)
    if at_payload:
        print("\nAccess token appears to be a JWT. Payload:")
        print(json.dumps(at_payload, indent=2))
    else:
        print("\nAccess token appears opaque (cannot decode as JWT).")

    # 3) Call a protected API endpoint with the access token
    print("\nCalling protected test endpoint with access token...")
    headers = {"Authorization": f"Bearer {access_token}"}
    try:
        api_resp = requests.get(TEST_API_URL, headers=headers, timeout=10, verify=False)
        print("Test endpoint status:", api_resp.status_code)
        try:
            print(api_resp.json())
        except Exception:
            print(api_resp.text)
    except Exception as e:
        print("Error calling test endpoint:", e)


if __name__ == "__main__":
    main()


