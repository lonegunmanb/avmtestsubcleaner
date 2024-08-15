package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/lonegunmanb/oneesrunnerscleaner/pkg"
)

var runnerNameRegex = regexp.MustCompile("[a-z0-9_]{15}")

func main() {
	subscriptionId := readEssentialEnv("AZURE_SUBSCRIPTION_ID")
	tenantId := readEssentialEnv("AZURE_TENANT_ID")
	ctx := context.Background()
	client, err := pkg.NewClient(subscriptionId, tenantId, ctx)
	if err != nil {
		panic(err.Error())
	}
	purgeRunnerPools(err, client)
	purgeResidualResourceGroups(client)
}

func purgeResidualResourceGroups(client *pkg.Client) {
	recordRg, err := client.EnsureResidualCleanerResourceGroup()
	if err != nil {
		panic(err.Error())
	}
	groups, err := client.ListAllResourceGroups()
	if err != nil {
		panic(err.Error())
	}
	wg := sync.WaitGroup{}

	for _, rg := range groups {
		if rg.IsProtected() {
			continue
		}
		if _, ok := recordRg.Tags[rg.Name]; ok {
			wg.Add(1)
			go func() {
				fmt.Printf("deleting resource group %s\n", rg.Name)
				defer wg.Done()
				err = client.DeleteResourceGroup(rg.Name)
				if err != nil {
					fmt.Printf("cannot delete resource group %s: %+v\n", rg.Name, err)
				} else {
					fmt.Printf("resource group %s deleted\n", rg.Name)
				}
			}()
			delete(recordRg.Tags, rg.Name)
			continue
		}
		if len(recordRg.Tags) <= 50 {
			recordRg.Tags[rg.Name] = strconv.FormatInt(time.Now().Unix(), 10)
		}
	}
	err = client.UpgradeResidualResourceGroupTags(recordRg)
	if err != nil {
		panic(err.Error())
	}
	wg.Wait()
}

func purgeRunnerPools(err error, client *pkg.Client) {
	pools, err := client.ListPools()
	if err != nil {
		panic(fmt.Sprintf("cannot list pools: %+v", err))
	}
	fmt.Printf("List pools, got %d pools\n", len(pools.Data))
	for _, pool := range pools.Data {
		var runnerNames []string
		fmt.Printf("Get runners for %s/%s\n", pool.ResourceGroup, pool.Name)
		runners, err := client.GetRunners(pool.ResourceGroup, pool.Name)
		if err != nil {
			println(fmt.Sprintf("cannot get runners for %s/%s: %+v", pool.ResourceGroup, pool.Name, err))
			continue
		}
		for _, r := range runners {
			if r.Status != "Allocated" {
				continue
			}
			if _, seen := pool.Tags[r.Name]; !seen {
				fmt.Printf("  unseen %s\n", r.Name)
				runnerNames = append(runnerNames, r.Name)
				continue
			}
			fmt.Printf("  purge runner %s from %s/%s\n", r.Id, pool.ResourceGroup, pool.Name)
			err = client.PurgeRunner(pool.ResourceGroup, pool.Name, r.Id)
			if err != nil {
				runnerNames = append(runnerNames, r.Name)
			}
		}
		tags := make(map[string]any)
		for key, value := range pool.Tags {
			if !runnerNameRegex.Match([]byte(key)) {
				tags[key] = value
			}
		}
		for _, n := range runnerNames {
			tags[n] = time.Now().UTC().String()
		}
		fmt.Printf("upgrade tags for %s/%s\n", pool.ResourceGroup, pool.Name)
		_ = client.UpgradePoolTags(pool, tags)
	}
}

func readEssentialEnv(envName string) string {
	r := os.Getenv(envName)
	if r == "" {
		panic(fmt.Sprintf("to run this test you must set env %s first", envName))
	}
	return r
}
