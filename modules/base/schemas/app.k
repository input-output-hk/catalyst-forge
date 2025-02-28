import k8s.apimachinery.pkg.apis.meta.v1

schema App:
    env: str
    metadata: v1.ObjectMeta
    selector: v1.LabelSelector