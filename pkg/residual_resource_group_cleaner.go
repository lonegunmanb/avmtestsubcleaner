package pkg

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"strings"
)

const RG_API_VERSION = "2020-06-01"
const RecorderRgName = "residualrgrecorder"

type ResourceGroup struct {
	Id       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Location string            `json:"location"`
	Tags     map[string]string `json:"tags"`
}

var protectedNameHashes = map[string]struct{}{
	"7e8e0b947214e31d9f02a94090487c0d": {},
	"cdb5195b6e59c8d4743d60fe5d1ddd54": {},
	"f054bc18556dfd9d4cd5c6f92a3b96db": {},
	"e35386fdf3abd880dd3b57c7bcb2340f": {},
	"62e5e9c30b987e3d11dbeb7a1b07ff70": {},
	"6e0030125d834f2263ff441f6b8d3ff7": {},
}

func (rg ResourceGroup) IsProtected() bool {
	if strings.HasPrefix(rg.Name, "MC_") {
		return true
	}
	if rg.Name == RecorderRgName {
		return true
	}
	if _, ok := protectedNameHashes[md5Hash(rg.Name)]; ok {
		return true
	}
	if _, preserve := rg.Tags["do_not_delete"]; preserve {
		return true
	}
	return false
}

func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func (c *Client) EnsureResidualCleanerResourceGroup() (ResourceGroup, error) {
	rgId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.SubscriptionId, RecorderRgName)
	rg := ResourceGroup{
		Location: "eastus",
	}
	_, err := c.resourceClient.Get(c.ctx, rgId, RG_API_VERSION, &rg)
	if err == nil {
		return rg, nil
	}
	if !strings.Contains(err.Error(), "ResourceGroupNotFound") {
		return rg, fmt.Errorf("cannot read resource group %s: %+v", rgId, err)
	}
	if _, err = c.resourceClient.CreateOrUpdate(c.ctx, rgId, RG_API_VERSION, &rg); err != nil {
		return rg, fmt.Errorf("cannot create resource group %s: %+v", rgId, err)
	}
	return rg, nil
}

func (c *Client) ListAllResourceGroups() ([]ResourceGroup, error) {
	rgns := fmt.Sprintf("/subscriptions/%s/resourceGroups/", c.SubscriptionId)
	list, err := c.resourceClient.List(c.ctx, rgns, RG_API_VERSION)
	if err != nil {
		return nil, fmt.Errorf("cannot list resource groups: %+v", err)
	}
	var result []ResourceGroup
	for _, value := range list.(map[string]any) {
		items := value.([]any)
		for _, item := range items {
			r := item.(map[string]any)
			rg := ResourceGroup{
				Id:       r["id"].(string),
				Name:     r["name"].(string),
				Type:     r["type"].(string),
				Location: r["location"].(string),
				Tags:     make(map[string]string),
			}
			tags := r["tags"]
			if tags != nil {
				for key, value := range tags.(map[string]any) {
					rg.Tags[key] = value.(string)
				}
			}
			result = append(result, rg)
		}
	}
	return result, nil
}

func (c *Client) UpgradeResidualResourceGroupTags(rg ResourceGroup) error {
	_, err := c.resourceClient.Action(c.ctx, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.SubscriptionId, rg.Name), "", RG_API_VERSION, "PATCH", nil, map[string]any{
		"tags": rg.Tags,
	})
	if err != nil {
		return fmt.Errorf("cannot update resource group %s: %+v", rg.Name, err)
	}
	return nil
}

func (c *Client) DeleteResourceGroup(name string) error {
	_, err := c.resourceClient.Delete(c.ctx, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.SubscriptionId, name), RG_API_VERSION)
	if err != nil {
		return fmt.Errorf("cannot delete resource group %s: %+v", name, err)
	}
	return nil
}

type Client struct {
	SubscriptionId string
	TenantId       string
	resourceClient *ResourceClient
	ctx            context.Context
	graphClient    *armresourcegraph.Client
}

func NewClient(subscriptionId, tenantId string, ctx context.Context) (*Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{
		TenantID: tenantId,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create credential: %+v", err)
	}

	resCli, err := NewResourceClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create resource client: %+v", err)
	}
	graphCli, err := armresourcegraph.NewClient(cred, &arm.ClientOptions{
		DisableRPRegistration: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create graph client: %+v", err)
	}
	c := &Client{
		SubscriptionId: subscriptionId,
		TenantId:       tenantId,
		resourceClient: resCli,
		graphClient:    graphCli,
		ctx:            ctx,
	}
	return c, nil
}
