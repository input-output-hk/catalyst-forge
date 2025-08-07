import k8s.api.apps.v1 as apps

# Get deployment configuration from option
deployment_config = option("deployment", default={
    "env": "dev",
    "instance": "test",
    "name": "simple",
    "namespace": "default",  
    "values": {
        "image": "nginx:1.20",
        "replicas": 1,
        "port": 80
    },
    "version": "1.0.0"
})

# Create deployment based on config
deployment = apps.Deployment {
    metadata = {
        name = deployment_config.instance
        namespace = deployment_config.namespace
        labels = {
            "app.kubernetes.io/name" = deployment_config.instance
            "app.kubernetes.io/managed-by" = "forge"
            "forge.projectcatalyst.io/name" = deployment_config.name
            "forge.projectcatalyst.io/version" = deployment_config.version
        }
    }
    spec = {
        replicas = deployment_config.values.replicas
        selector = {
            matchLabels = {
                "app.kubernetes.io/name" = deployment_config.instance
            }
        }
        template = {
            metadata = {
                labels = {
                    "app.kubernetes.io/name" = deployment_config.instance
                    "app.kubernetes.io/managed-by" = "forge"
                }
            }
            spec = {
                containers = [{
                    name = "main"
                    image = deployment_config.values.image
                    ports = [{
                        containerPort = deployment_config.values.port
                    }]
                }]
            }
        }
    }
}

# Output YAML
manifests.yaml_stream([deployment])