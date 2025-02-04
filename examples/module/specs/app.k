import base.schemas
import k8s.apimachinery.pkg.apis.meta.v1

schema App(schemas.App):
    metadata: v1.ObjectMeta
    image: str
    replicas: int = 1
    port: int