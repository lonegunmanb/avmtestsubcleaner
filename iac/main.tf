terraform {
  required_version = ">= 1.2"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.11, < 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0.0"
    }
  }
}

provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

resource "azurerm_resource_group" "example" {
  name     = "residual_resource_group_cleaner"
  location = "westeurope"
}

resource "azurerm_log_analytics_workspace" "example" {
  name                = "cleanerlog"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_container_app_environment" "example" {
  name                       = "cleaner"
  location                   = azurerm_resource_group.example.location
  resource_group_name        = azurerm_resource_group.example.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.example.id
}

data "azurerm_client_config" "this" {}

resource "azurerm_container_app_job" "example" {
  name                         = "cleaner"
  location                     = azurerm_resource_group.example.location
  resource_group_name          = azurerm_resource_group.example.name
  container_app_environment_id = azurerm_container_app_environment.example.id

  replica_timeout_in_seconds = 3600
  replica_retry_limit        = 10
  schedule_trigger_config {
    cron_expression          = "0 10,22 * * *"
    parallelism              = 1
    replica_completion_count = 1
  }

  template {
    container {
      image   = "mcr.microsoft.com/azterraform"
      name    = "cleaner"
      command = ["/bin/bash"]
      args = [
        "-c",
        "git clone https://github.com/lonegunmanb/avmtestsubcleaner.git && cd avmtestsubcleaner && go run main.go",
      ]

      cpu    = 1
      memory = "2Gi"
      env {
        name        = "ARM_SUBSCRIPTION_ID"
        secret_name = "azure-subscription-id"
      }
      env {
        name        = "AZURE_SUBSCRIPTION_ID"
        secret_name = "azure-subscription-id"
      }
      env {
        name        = "ARM_TENANT_ID"
        secret_name = "azure-tenant-id"
      }
      env {
        name        = "AZURE_TENANT_ID"
        secret_name = "azure-tenant-id"
      }
      env {
        name        = "ARM_CLIENT_ID"
        secret_name = "azure-client-id"
      }
      env {
        name        = "AZURE_CLIENT_ID"
        secret_name = "azure-client-id"
      }
      env {
        name        = "AZURE_CLIENT_SECRET"
        secret_name = "azure-client-secret"
      }
      env {
        name        = "ARM_CLIENT_SECRET"
        secret_name = "azure-client-secret"
      }
    }
  }
  secret {
    name  = "azure-client-id"
    value = var.client_id
  }
  secret {
    name  = "azure-subscription-id"
    value = data.azurerm_client_config.this.subscription_id
  }
  secret {
    name  = "azure-tenant-id"
    value = data.azurerm_client_config.this.tenant_id
  }
  secret {
    name  = "azure-client-secret"
    value = var.client_secret
  }
}