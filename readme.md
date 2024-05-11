# Azure Runner Pools Manager

This program is a utility for managing 1ES hosted Runner Pools. It's written in Go and uses the Azure SDK for Go to interact with Azure services.

## Why we need this tool?

It looks like 1ES runner pool would provision a runner for a corresponding GitHub action job, as expected, but when this job requires a manual approval to run, and it would not be approved, this pending runner would stick at `Allocated` status and cause meaningless resource waste. This tool is a cronjob to clean these "zombie" runners regularly.

## Main Functionality

1. **Environment Variables**: The program reads Azure Subscription ID and Tenant ID from environment variables.

2. **Listing Pools**: It lists all the Runner Pools available in the Azure subscription.

3. **Processing Runners**: For each Runner Pool, it retrieves the runners and checks their status. If a runner is not allocated or not seen before, it's added to a list. If a runner is allocated and seen before, it's purged from the pool.

4. **Updating Pool Tags**: After processing all runners in a pool, it updates the pool's tags with the list of unseen runners and the current timestamp.
