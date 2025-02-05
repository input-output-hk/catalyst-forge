import file
import yaml

import schemas

import k8s.apimachinery.pkg.apis.meta.v1

Read = lambda -> any {
    _deployment: any = option("deployment", default=None)
    _deploymentFile: str = option("deployment-file", default=None)

    assert _deployment != None or _deploymentFile != None, "deployment or deployment-file must be provided"

    if _deployment != None:
        _d = schemas.Deployment { **_deployment }
    else:
        _d = schemas.Deployment {
            **yaml.decode(file.read(_deploymentFile))
        }

    _d.values | {
        metadata = v1.ObjectMeta {
            name: _d.instance
            namespace: _d.namespace
            labels: {
                "app.kubernetes.io/managed-by": "forge"
                "app.kubernetes.io/name": _d.instance
                "forge.projectcatalyst.io/name": _d.name
                "forge.projectcatalyst.io/version": _d.version
            }
        }

        selector =  v1.LabelSelector {
            matchLabels: {
                "app.kubernetes.io/name": _d.instance
            }
        }
    }
}