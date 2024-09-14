package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lonegunmanb/oneesrunnerscleaner/pkg"
)

func main() {
	subscriptionId := readEssentialEnv("AZURE_SUBSCRIPTION_ID")
	tenantId := readEssentialEnv("AZURE_TENANT_ID")
	ctx := context.Background()
	client, err := pkg.NewClient(subscriptionId, tenantId, ctx)
	if err != nil {
		panic(err.Error())
	}
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

	existingRgs := make(map[string]struct{})
	for _, rg := range groups {
		existingRgs[rg.Name] = struct{}{}
	}
	var deprecatedRgs []string
	for k, _ := range recordRg.Tags {
		if _, ok := existingRgs[k]; !ok {
			deprecatedRgs = append(deprecatedRgs, k)
		}
	}
	for _, rg := range deprecatedRgs {
		delete(recordRg.Tags, rg)
	}

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
		if len(recordRg.Tags) < 50 {
			recordRg.Tags[rg.Name] = strconv.FormatInt(time.Now().Unix(), 10)
		}
	}
	err = client.UpgradeResidualResourceGroupTags(recordRg)
	if err != nil {
		panic(err.Error())
	}
	wg.Wait()
}

func readEssentialEnv(envName string) string {
	r := os.Getenv(envName)
	if r == "" {
		panic(fmt.Sprintf("to run this test you must set env %s first", envName))
	}
	return r
}
