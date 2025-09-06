"""
Configuration models for DNS task.
"""

from pydantic import BaseModel


class DNSConfig(BaseModel):
    """Configuration for DNS setup and CoreDNS wildcard configuration.

    This model defines parameters for configuring wildcard DNS resolution
    that routes all subdomains to the Envoy gateway for ingress routing.
    """

    domain: str = "projectcatalyst.dev"
    """The root domain for wildcard DNS configuration.

    All subdomains of this domain (*.domain) will be routed to the
    Envoy gateway ClusterIP for ingress processing.
    Default: "projectcatalyst.dev"
    """

    coredns_namespace: str = "kube-system"
    """Kubernetes namespace where CoreDNS is running.

    CoreDNS typically runs in kube-system in most Kubernetes distributions.
    Default: "kube-system"
    """

    coredns_configmap: str = "coredns"
    """Name of the CoreDNS ConfigMap containing the Corefile configuration.

    This ConfigMap contains the DNS server configuration that will be
    modified to add wildcard DNS entries.
    Default: "coredns"
    """
