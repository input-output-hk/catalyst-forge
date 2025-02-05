import k8s.apimachinery.pkg.apis.meta.v1

schema App:
    # metadata = v1.ObjectMeta {
    #     name: d.instance
    #     namespace: d.namespace
    #     labels: {
    #         "app.kubernetes.io/name": d.instance
    #     }
    # }
    # selector =  v1.LabelSelector {
    #     matchLabels: {
    #         "app.kubernetes.io/name": d.instance
    #     }
    # }
    metadata: v1.ObjectMeta
    selector: v1.LabelSelector