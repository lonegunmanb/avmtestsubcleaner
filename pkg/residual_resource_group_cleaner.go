package pkg

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
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
	"403c2d5ed4f3bd5390c33e9a0b38e2ff": {},
	"45359b60f526c42dc5f169b79183a782": {},
	"7e8e0b947214e31d9f02a94090487c0d": {},
	"cdb5195b6e59c8d4743d60fe5d1ddd54": {},
	"f054bc18556dfd9d4cd5c6f92a3b96db": {},
	"e35386fdf3abd880dd3b57c7bcb2340f": {},
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
