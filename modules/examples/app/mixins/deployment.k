import protocols

import base.schemas
import k8s.api.apps.v1 as appsv1

mixin DeploymentMixin for protocols.AppProtocol:
    manifests: schemas.Manifest {
        Deployment = appsv1.Deployment {
            metadata = app.metadata
            spec = {
                replicas = app.replicas
                selector = {
                    matchLabels: {
                        "app.kubernetes.io/name": app.metadata.name
                    }
                }
                template = {
                    metadata = app.metadata
                    spec = {
                        containers = [
                            {
                                name: "main"
                                image: app.image
                                ports: [
                                    {
                                        containerPort: app.port
                                    }
                                ]
                            }
                        ]
                    }
                }
            }
        }
    }