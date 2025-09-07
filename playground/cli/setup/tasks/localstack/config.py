"""
Configuration models for localstack task.
"""

from pydantic import BaseModel


class LocalStackConfig(BaseModel):
    """Configuration for LocalStack AWS service simulation deployment.

    This model defines all the parameters needed to deploy and configure
    LocalStack for AWS service simulation in the playground environment.
    """

    namespace: str = "localstack"
    """Kubernetes namespace where LocalStack will be deployed.

    All LocalStack components will be installed in this namespace.
    Default: "localstack"
    """

    chart: str = "localstack/localstack"
    """Helm chart to use for LocalStack deployment.

    Specifies the Helm chart repository and name for LocalStack.
    Default: "localstack/localstack"
    """

    services: str = "s3,secretsmanager,ssm,sts,iam,dynamodb,sqs,sns"
    """Comma-separated list of AWS services to enable in LocalStack.

    These services will be started and available for use.
    Default: "s3,secretsmanager,ssm,sts,iam,dynamodb,sqs,sns"
    """

    service_type: str = "ClusterIP"
    """Kubernetes service type for LocalStack.

    Determines how the LocalStack service is exposed.
    Options: ClusterIP, LoadBalancer, NodePort
    Default: "ClusterIP"
    """

    persistence: dict = {"enabled": True, "size": "5Gi", "storage_class": "local-path"}
    """Persistence configuration for LocalStack data.

    Controls whether data is stored persistently and the storage configuration.
    """

    resources: dict = {"requests": {"cpu": "100m", "memory": "256Mi"}, "limits": {"memory": "1Gi"}}
    """Resource requests and limits for LocalStack pods.

    Defines CPU and memory allocations for LocalStack containers.
    """

    region: str = "us-east-1"
    """Default AWS region for LocalStack.

    This region will be used as the default for AWS service calls.
    Default: "us-east-1"
    """

    aws_access_key: str = "test"
    """AWS access key ID for LocalStack authentication.

    This is the access key that will be used for AWS API calls.
    Default: "test"
    """

    aws_secret_key: str = "test"
    """AWS secret access key for LocalStack authentication.

    This is the secret key that will be used for AWS API calls.
    Default: "test"
    """

    debug: bool = False
    """Whether to enable debug logging for LocalStack.

    When enabled, LocalStack will output detailed debug information.
    Default: False
    """
