"""
HTTP operations helper with retry logic.
"""

import requests
from typing import Dict, Any, IO, Optional
from time import sleep


class HTTP:
    """Helper for HTTP operations with retry logic."""

    def wait_for_ready(
        self, url: str, log: IO, timeout: int = 300, interval: int = 5, expected_status: int = 200
    ) -> bool:
        """
        Wait for an HTTP endpoint to be ready.

        Args:
            url: URL to check
            log: Log file handle
            timeout: Maximum time to wait in seconds
            interval: Time between checks in seconds
            expected_status: Expected HTTP status code

        Returns:
            True if endpoint is ready, False if timeout
        """
        log.write(f"Waiting for {url} to be ready (expecting {expected_status})...\n")
        log.flush()

        elapsed = 0
        while elapsed < timeout:
            try:
                response = requests.get(url, timeout=5, verify=False)
                if response.status_code == expected_status:
                    log.write(f"✓ {url} is ready (status: {response.status_code})\n")
                    log.flush()
                    return True
                else:
                    log.write(f"  Status: {response.status_code} (waiting for {expected_status})\n")
            except requests.RequestException as e:
                log.write(f"  Connection error: {e}\n")

            log.flush()
            sleep(interval)
            elapsed += interval

        log.write(f"✗ Timeout waiting for {url}\n")
        log.flush()
        return False

    def get(self, url: str, **kwargs) -> requests.Response:
        """
        Simple GET request.

        Args:
            url: URL to request
            **kwargs: Additional arguments for requests.get

        Returns:
            Response object
        """
        return requests.get(url, **kwargs)

    def post_json(
        self,
        url: str,
        data: Dict[str, Any],
        headers: Optional[Dict[str, str]] = None,
        auth: Optional[tuple] = None,
        verify: bool = True,
    ) -> requests.Response:
        """
        POST JSON data to an endpoint.

        Args:
            url: Target URL
            data: Data dictionary to send as JSON
            headers: Optional headers
            auth: Optional auth tuple (username, password)
            verify: Whether to verify SSL certificates

        Returns:
            Response object
        """
        default_headers = {"Content-Type": "application/json"}
        if headers:
            default_headers.update(headers)

        response = requests.post(url, json=data, headers=default_headers, auth=auth, verify=verify)
        response.raise_for_status()
        return response


# Singleton instance
http = HTTP()
