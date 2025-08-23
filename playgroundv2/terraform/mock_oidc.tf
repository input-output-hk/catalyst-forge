locals {
  mock_oidc_labels = {
    app = "mock-oidc"
  }
}

resource "kubernetes_namespace_v1" "mock_oidc" {
  metadata {
    name = var.playground.mock_oidc.namespace
  }
}

resource "kubernetes_deployment_v1" "mock_oidc" {
  metadata {
    name      = "mock-oidc"
    namespace = var.playground.mock_oidc.namespace
    labels    = local.mock_oidc_labels
  }
  spec {
    replicas = 1

    selector {
      match_labels = local.mock_oidc_labels
    }

    template {
      metadata {
        labels = local.mock_oidc_labels
      }
      spec {
        container {
          name  = "mock-oauth2-server"
          image = var.playground.mock_oidc.image

          port {
            container_port = var.playground.mock_oidc.service_port
          }

          env {
            name  = "SERVER_PORT"
            value = tostring(var.playground.mock_oidc.service_port)
          }

          # Optional: enable simple interactive login page
          env {
            name  = "INTERACTIVE_LOGIN"
            value = "true"
          }

          # Configure a static issuer so discovery works behind Envoy/HTTPRoute
          env {
            name  = "TOKEN_ENDPOINT_AUTH_METHOD"
            value = "client_secret_basic"
          }

          # JSON config contains client and issuer aliases
          env {
            name = "JSON_CONFIG"
            value = jsonencode({
              interactiveLogin = true,
              httpServer       = { port = var.playground.mock_oidc.service_port },
              tokenCallbacks   = [],
              # Default issuer (path segment) used in endpoints: /.well-known/openid-configuration and /jwks
              # The server supports multi-tenancy by path; use "/default" to keep it simple.
              # Clients must be configured for this issuerId.
              issuers = [
                {
                  issuerId  = "default",
                  audiences = ["kratos"],
                  cookie    = { secureCookie = false },
                  clients = [
                    {
                      clientId     = var.playground.mock_oidc.client_id,
                      clientSecret = var.playground.mock_oidc.client_secret,
                      redirectUris = [
                        # Kratos callback path for provider id "google"
                        "https://${var.playground.kratos.host}/.ory/kratos/public/self-service/methods/oidc/callback/google"
                      ],
                      scopes                  = ["openid", "email", "profile"],
                      accessTokenTTL          = "PT1H",
                      idTokenTTL              = "PT1H",
                      tokenEndpointAuthMethod = "client_secret_basic"
                    }
                  ]
                }
              ]
            })
          }

          # Make it easy to see logs while testing
          resources {
            limits = {
              cpu    = "200m"
              memory = "256Mi"
            }
            requests = {
              cpu    = "50m"
              memory = "64Mi"
            }
          }
        }
      }
    }
  }
}

resource "kubernetes_service_v1" "mock_oidc" {
  metadata {
    name      = "mock-oidc"
    namespace = var.playground.mock_oidc.namespace
    labels    = local.mock_oidc_labels
  }
  spec {
    selector = local.mock_oidc_labels

    port {
      name        = "http"
      port        = 80
      target_port = var.playground.mock_oidc.service_port
    }
  }
}

resource "kubectl_manifest" "mock_oidc_route" {
  depends_on = [kubernetes_service_v1.mock_oidc, kubectl_manifest.gateway]

  yaml_body = templatefile("${path.module}/templates/mock-oidc-route.yaml.tftpl", {
    mock_oidc_namespace = var.playground.mock_oidc.namespace
    gateway_namespace   = local.envoy_gateway_namespace
    gateway_name        = var.playground.envoy.gateway_name
    hostname            = var.playground.mock_oidc.host
    service_port        = 80
  })
}


