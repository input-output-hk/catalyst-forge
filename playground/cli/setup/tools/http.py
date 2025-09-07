"""
HTTP operations helper with retry logic.
"""

import requests


class HTTP:
    """Helper for HTTP operations with retry logic."""

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


# Singleton instance
http = HTTP()
