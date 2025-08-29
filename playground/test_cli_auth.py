import hashlib
import base64
import webbrowser
import socket
from http.server import BaseHTTPRequestHandler, HTTPServer
import secrets
import time
import requests

# --- Configuration ---
HYDRA_PUBLIC_URL = "https://auth.projectcatalyst.dev/hydra/public"
CLIENT_ID = "forge-cli"
REDIRECT_URI = "http://127.0.0.1:49152/callback"
SCOPE = "openid offline"

# --- PKCE Code Generation ---
def generate_pkce_codes():
    code_verifier = secrets.token_urlsafe(64)
    code_challenge = hashlib.sha256(code_verifier.encode("utf-8")).digest()
    code_challenge = base64.urlsafe_b64encode(code_challenge).decode("utf-8").replace("=", "")
    return code_verifier, code_challenge

# --- HTTP Server for Callback ---
class CallbackHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/callback"):
            self.send_response(200)
            self.send_header("Content-type", "text/html")
            self.end_headers()
            self.wfile.write(b"Authentication successful! You can close this window.")
            global authorization_code, received_state
            authorization_code = self.path.split("code=")[1].split("&")[0]
            received_state = self.path.split("state=")[1].split("&")[0]
            return

def run_server(server_class=HTTPServer, handler_class=CallbackHandler, port=49152):
    server_address = ("", port)
    httpd = server_class(server_address, handler_class)
    httpd.handle_request()

# --- Main Authentication Flow ---
if __name__ == "__main__":
    code_verifier, code_challenge = generate_pkce_codes()
    state = secrets.token_urlsafe(16)

    # 1. Construct Authorization URL
    auth_url = (
        f"{HYDRA_PUBLIC_URL}/oauth2/auth?"
        f"client_id={CLIENT_ID}&"
        f"response_type=code&"
        f"scope={SCOPE}&"
        f"redirect_uri={REDIRECT_URI}&"
        f"code_challenge={code_challenge}&"
        f"code_challenge_method=S256&"
        f"state={state}"
    )

    # 2. Open Browser and Start Server
    print("Opening browser for authentication...")
    time.sleep(2)
    print(auth_url)
    #webbrowser.open(auth_url)
    authorization_code = None
    received_state = None
    run_server()

    if authorization_code and received_state == state:
        print(f"Authorization code received: {authorization_code}")

        # 3. Exchange Authorization Code for Access Token
        token_url = f"{HYDRA_PUBLIC_URL}/oauth2/token"
        token_data = {
            "grant_type": "authorization_code",
            "code": authorization_code,
            "redirect_uri": REDIRECT_URI,
            "client_id": CLIENT_ID,
            "code_verifier": code_verifier,
        }
        response = requests.post(token_url, data=token_data, verify=False)

        if response.status_code == 200:
            print("Access token received:")
            print(response.json())
        else:
            print("Error getting access token:")
            print(response.status_code, response.text)
            print("Response headers:")
            for header, value in response.headers.items():
                print(f"{header}: {value}")
    else:
        print("Failed to get authorization code or state mismatch.")