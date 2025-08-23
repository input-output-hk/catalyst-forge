terraform {
  required_version = ">= 1.5.0"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.32"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "~> 1.14"
    }
  }
}

provider "kubernetes" {
  config_path = var.playground.global.kubeconfig_path
}

provider "helm" {
  kubernetes {
    config_path = var.playground.global.kubeconfig_path
  }
}

provider "kubectl" {
  config_path = var.playground.global.kubeconfig_path
}