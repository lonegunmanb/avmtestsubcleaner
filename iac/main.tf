resource azurerm_resource_group example {
  name = "residual_resource_group_cleaner"
  location = "westeurope"
}

resource "azurerm_log_analytics_workspace" "example" {
  name                = "example-log-analytics-workspace"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_container_app_environment" "example" {
  name                = "cleaner"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.example.id
}

resource "azurerm_user_assigned_identity" "identity" {
  location            = azurerm_resource_group.example.location
  name                = "cleaner"
  resource_group_name = azurerm_resource_group.example.name
}

data "azurerm_client_config" "this" {}

resource "azurerm_role_assignment" "assignment" {
  principal_id         = azurerm_user_assigned_identity.identity.principal_id
  scope                = "/subscriptions/${data.azurerm_client_config.this.subscription_id}"
  role_definition_name = "Owner"
}

resource "azurerm_container_app_job" "example" {
  name                         = "cleaner"
  location                     = azurerm_resource_group.example.location
  resource_group_name          = azurerm_resource_group.example.name
  container_app_environment_id = azurerm_container_app_environment.example.id

  replica_timeout_in_seconds = 10
  replica_retry_limit        = 10
  manual_trigger_config {
    parallelism              = 1
    replica_completion_count = 1
  }
  #   schedule_trigger_config {
  #     cron_expression = "0 10,22 * * *"
  #     parallelism              = 1
  #     replica_completion_count = 1
  #   }

  template {
    container {
      image = "mcr.microsoft.com/azterraform"
      name  = "cleaner"
      command = ["bash"]
      args = ["-c", "\"git clone https://github.com/lonegunmanb/avmtestsubcleaner.git && cd avmtestsubcleaner && az login --identity --username $MSI_ID > /dev/null && go run main.go\""]

      cpu    = 1
      memory = "2Gi"
      env {
        name = "ARM_SUBSCRIPTION_ID"
        secret_name = "azure-subscription-id"
      }
      env {
        name = "ARM_TENANT_ID"
        secret_name = "azure-tenant-id"
      }
      env {
        name = "ARM_CLIENT_ID"
        secret_name = "azure-client-id"
      }
      env {
        name = "ARM_USE_MSI"
        value = "true"
      }
    }
  }
  secret {
    name  = "azure-client-id"
    value = azurerm_user_assigned_identity.identity.client_id
  }
  secret {
    name  = "azure-subscription-id"
    value = data.azurerm_client_config.this.subscription_id
  }
  secret {
    name  = "azure-tenant-id"
    value = data.azurerm_client_config.this.tenant_id
  }
  identity {
    type = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.identity.id]
  }
}