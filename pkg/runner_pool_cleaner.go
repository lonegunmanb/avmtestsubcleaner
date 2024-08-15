package pkg

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"net/http"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
)

var runnerNameRegex = regexp.MustCompile("([a-zA-Z0-9_-]){15}")

const API_VERSION = "2020-05-07"

type RunnerPool struct {
	Id       string            `json:"id"`
	Name     string            `json:"name"`
	Location string            `json:"location"`
	Tags     map[string]string `json:"tags"`
}

type Runner struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Client struct {
	SubscriptionId string
	TenantId       string
	resourceClient *ResourceClient
	ctx            context.Context
	graphClient    *armresourcegraph.Client
}

type Pool struct {
	Name          string         `json:"name"`
	ResourceGroup string         `json:"resourceGroup"`
	Location      string         `json:"location"`
	Id            string         `json:"id"`
	Tags          map[string]any `json:"tags"`
}

type PoolsResponse struct {
	Data []Pool
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

func (c *Client) UpgradePoolTags(pool Pool, tags map[string]any) error {
	_, err := c.resourceClient.Action(c.ctx, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CloudTest/hostedpools/%s", c.SubscriptionId, pool.ResourceGroup, pool.Name), "", API_VERSION, "PATCH", nil, map[string]any{
		"tags": tags,
	})
	if err != nil {
		return fmt.Errorf("cannot update pool tags for %s/%s: %+v", pool.ResourceGroup, pool.Name, err)
	}
	return nil
}

func (c *Client) GetRunnerPool(resourceGroupName, poolName string) (*RunnerPool, error) {
	resp, err := c.resourceClient.Get(c.ctx, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CloudTest/hostedpools/%s", c.SubscriptionId, resourceGroupName, poolName), API_VERSION, &RunnerPool{})
	if err != nil {
		return nil, fmt.Errorf("cannot get pool: %+v", err)
	}
	return resp.(*RunnerPool), nil
}

func (c *Client) GetRunners(resourceGroupName, poolName string) ([]*Runner, error) {
	var runners []*Runner
	_, err := c.resourceClient.Get(c.ctx, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CloudTest/hostedpools/%s/resources", c.SubscriptionId, resourceGroupName, poolName), API_VERSION, &runners)
	if err != nil {
		return nil, fmt.Errorf("cannot get pool: %+v", err)
	}
	return runners, nil
}

func (c *Client) PurgeRunner(resourceGroupName, poolName, runnerId string) error {
	_, err := c.resourceClient.Action(c.ctx, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CloudTest/hostedpools/%s/resources", c.SubscriptionId, resourceGroupName, poolName), "", API_VERSION, "DELETE", &http.Header{
		"commandName": []string{"Microsoft_Azure_CloudTest."},
	}, []string{runnerId})
	return err
}

func (c *Client) ListPools() (*PoolsResponse, error) {
	resp := &PoolsResponse{}
	options := &armresourcegraph.QueryRequestOptions{
		Skip:         p(int32(0)),
		Top:          p(int32(100)),
		ResultFormat: p(armresourcegraph.ResultFormatObjectArray),
	}
	for {
		response, err := c.graphClient.Resources(
			c.ctx,
			armresourcegraph.QueryRequest{
				Query:            p("resources|where type =~ 'microsoft.cloudtest/hostedpools'\r\n| extend organizationType=parse_json(properties).organizationProfile.type\r\n| project id, name, type, location, subscriptionId, resourceGroup, tags, kind, organizationType|extend locationDisplayName=case(location =~ 'eastus','East US',location =~ 'eastus2','East US 2',location =~ 'southcentralus','South Central US',location =~ 'westus2','West US 2',location =~ 'westus3','West US 3',location =~ 'australiaeast','Australia East',location =~ 'southeastasia','Southeast Asia',location =~ 'northeurope','North Europe',location =~ 'swedencentral','Sweden Central',location =~ 'uksouth','UK South',location =~ 'westeurope','West Europe',location =~ 'centralus','Central US',location =~ 'southafricanorth','South Africa North',location =~ 'centralindia','Central India',location =~ 'eastasia','East Asia',location =~ 'japaneast','Japan East',location =~ 'koreacentral','Korea Central',location =~ 'canadacentral','Canada Central',location =~ 'francecentral','France Central',location =~ 'germanywestcentral','Germany West Central',location =~ 'italynorth','Italy North',location =~ 'norwayeast','Norway East',location =~ 'polandcentral','Poland Central',location =~ 'switzerlandnorth','Switzerland North',location =~ 'mexicocentral','Mexico Central',location =~ 'uaenorth','UAE North',location =~ 'brazilsouth','Brazil South',location =~ 'israelcentral','Israel Central',location =~ 'qatarcentral','Qatar Central',location =~ 'centralusstage','Central US (Stage)',location =~ 'eastusstage','East US (Stage)',location =~ 'eastus2stage','East US 2 (Stage)',location =~ 'northcentralusstage','North Central US (Stage)',location =~ 'southcentralusstage','South Central US (Stage)',location =~ 'westusstage','West US (Stage)',location =~ 'westus2stage','West US 2 (Stage)',location =~ 'asia','Asia',location =~ 'asiapacific','Asia Pacific',location =~ 'australia','Australia',location =~ 'brazil','Brazil',location =~ 'canada','Canada',location =~ 'europe','Europe',location =~ 'france','France',location =~ 'germany','Germany',location =~ 'global','Global',location =~ 'india','India',location =~ 'israel','Israel',location =~ 'italy','Italy',location =~ 'japan','Japan',location =~ 'korea','Korea',location =~ 'newzealand','New Zealand',location =~ 'norway','Norway',location =~ 'poland','Poland',location =~ 'qatar','Qatar',location =~ 'singapore','Singapore',location =~ 'southafrica','South Africa',location =~ 'sweden','Sweden',location =~ 'switzerland','Switzerland',location =~ 'uae','United Arab Emirates',location =~ 'uk','United Kingdom',location =~ 'unitedstates','United States',location =~ 'unitedstateseuap','United States EUAP',location =~ 'eastasiastage','East Asia (Stage)',location =~ 'southeastasiastage','Southeast Asia (Stage)',location =~ 'brazilus','Brazil US',location =~ 'northcentralus','North Central US',location =~ 'westus','West US',location =~ 'japanwest','Japan West',location =~ 'jioindiawest','Jio India West',location =~ 'westcentralus','West Central US',location =~ 'southafricawest','South Africa West',location =~ 'australiacentral','Australia Central',location =~ 'australiacentral2','Australia Central 2',location =~ 'australiasoutheast','Australia Southeast',location =~ 'jioindiacentral','Jio India Central',location =~ 'koreasouth','Korea South',location =~ 'southindia','South India',location =~ 'westindia','West India',location =~ 'canadaeast','Canada East',location =~ 'francesouth','France South',location =~ 'germanynorth','Germany North',location =~ 'norwaywest','Norway West',location =~ 'switzerlandwest','Switzerland West',location =~ 'ukwest','UK West',location =~ 'uaecentral','UAE Central',location =~ 'brazilsoutheast','Brazil Southeast',location)|extend subscriptionDisplayName=case(subscriptionId =~ 'f7a632a5-49db-4c5e-9828-cd62cb753971','Azure Verified Module Test Subscription',subscriptionId)|where (type !~ ('dell.storage/filesystems'))|where (type !~ ('purestorage.block/storagepools/avsstoragecontainers'))|where (type !~ ('purestorage.block/reservations'))|where (type !~ ('purestorage.block/storagepools'))|where (type !~ ('solarwinds.observability/organizations'))|where (type !~ ('splitio.experimentation/experimentationworkspaces'))|where (type !~ ('microsoft.agfoodplatform/farmbeats'))|where (type !~ ('microsoft.network/networkmanagers/verifierworkspaces'))|where (type !~ ('microsoft.mobilepacketcore/networkfunctions'))|where (type !~ ('microsoft.cdn/profiles/customdomains'))|where (type !~ ('microsoft.cdn/profiles/afdendpoints'))|where (type !~ ('microsoft.cdn/profiles/origingroups/origins'))|where (type !~ ('microsoft.cdn/profiles/origingroups'))|where (type !~ ('microsoft.cdn/profiles/afdendpoints/routes'))|where (type !~ ('microsoft.cdn/profiles/rulesets/rules'))|where (type !~ ('microsoft.cdn/profiles/rulesets'))|where (type !~ ('microsoft.cdn/profiles/secrets'))|where (type !~ ('microsoft.cdn/profiles/securitypolicies'))|where (type !~ ('microsoft.sovereign/landingzoneconfigurations'))|where (type !~ ('microsoft.hardwaresecuritymodules/cloudhsmclusters'))|where (type !~ ('microsoft.compute/computefleetinstances'))|where (type !~ ('microsoft.compute/standbypoolinstance'))|where (type !~ ('microsoft.compute/virtualmachineflexinstances'))|where (type !~ ('microsoft.kubernetesconfiguration/extensions'))|where (type !~ ('microsoft.containerservice/managedclusters/microsoft.kubernetesconfiguration/extensions'))|where (type !~ ('microsoft.kubernetes/connectedclusters/microsoft.kubernetesconfiguration/namespaces'))|where (type !~ ('microsoft.containerservice/managedclusters/microsoft.kubernetesconfiguration/namespaces'))|where (type !~ ('microsoft.kubernetes/connectedclusters/microsoft.kubernetesconfiguration/fluxconfigurations'))|where (type !~ ('microsoft.containerservice/managedclusters/microsoft.kubernetesconfiguration/fluxconfigurations'))|where (type !~ ('microsoft.portalservices/extensions/deployments'))|where (type !~ ('microsoft.portalservices/extensions'))|where (type !~ ('microsoft.portalservices/extensions/slots'))|where (type !~ ('microsoft.portalservices/extensions/versions'))|where (type !~ ('microsoft.datacollaboration/workspaces'))|where (type !~ ('microsoft.deviceregistry/devices'))|where (type !~ ('microsoft.deviceupdate/updateaccounts/activedeployments'))|where (type !~ ('microsoft.deviceupdate/updateaccounts/agents'))|where (type !~ ('microsoft.deviceupdate/updateaccounts/deployments'))|where (type !~ ('microsoft.deviceupdate/updateaccounts/deviceclasses'))|where (type !~ ('microsoft.deviceupdate/updateaccounts/updates'))|where (type !~ ('microsoft.deviceupdate/updateaccounts'))|where (type !~ ('private.devtunnels/tunnelplans'))|where (type !~ ('microsoft.impact/connectors'))|where (type !~ ('microsoft.edgeorder/virtual_orderitems'))|where (type !~ ('microsoft.workloads/epicvirtualinstances'))|where (type !~ ('microsoft.fairfieldgardens/provisioningresources/provisioningpolicies'))|where (type !~ ('microsoft.fairfieldgardens/provisioningresources'))|where (type !~ ('microsoft.fileshares/fileshares'))|where (type !~ ('microsoft.healthmodel/healthmodels'))|where (type !~ ('microsoft.hybridcompute/arcserverwithwac'))|where (type !~ ('microsoft.hybridcompute/machinessovereign'))|where (type !~ ('microsoft.hybridcompute/machinesesu'))|where (type !~ ('microsoft.network/virtualhubs')) or ((kind =~ ('routeserver')))|where (type !~ ('microsoft.network/networkvirtualappliances'))|where (type !~ ('microsoft.modsimworkbench/workbenches/chambers'))|where (type !~ ('microsoft.modsimworkbench/workbenches/chambers/connectors'))|where (type !~ ('microsoft.modsimworkbench/workbenches/chambers/files'))|where (type !~ ('microsoft.modsimworkbench/workbenches/chambers/filerequests'))|where (type !~ ('microsoft.modsimworkbench/workbenches/chambers/licenses'))|where (type !~ ('microsoft.modsimworkbench/workbenches/chambers/storages'))|where (type !~ ('microsoft.modsimworkbench/workbenches/chambers/workloads'))|where (type !~ ('microsoft.modsimworkbench/workbenches/sharedstorages'))|where (type !~ ('microsoft.insights/diagnosticsettings'))|where not((type =~ ('microsoft.network/serviceendpointpolicies')) and ((kind =~ ('internal'))))|where (type !~ ('microsoft.resources/resourcegraphvisualizer'))|where (type !~ ('microsoft.openlogisticsplatform/workspaces'))|where (type !~ ('microsoft.iotoperationsmq/mq'))|where (type !~ ('microsoft.orbital/cloudaccessrouters'))|where (type !~ ('microsoft.orbital/terminals'))|where (type !~ ('microsoft.orbital/sdwancontrollers'))|where (type !~ ('microsoft.recommendationsservice/accounts/modeling'))|where (type !~ ('microsoft.recommendationsservice/accounts/serviceendpoints'))|where (type !~ ('microsoft.recoveryservicesbvtd/vaults'))|where (type !~ ('microsoft.recoveryservicesbvtd2/vaults'))|where (type !~ ('microsoft.recoveryservicesintd/vaults'))|where (type !~ ('microsoft.recoveryservicesintd2/vaults'))|where (type !~ ('microsoft.features/featureprovidernamespaces/featureconfigurations'))|where (type !~ ('microsoft.deploymentmanager/rollouts'))|where (type !~ ('microsoft.providerhub/providerregistrations'))|where (type !~ ('microsoft.providerhub/providerregistrations/customrollouts'))|where (type !~ ('microsoft.providerhub/providerregistrations/defaultrollouts'))|where (type !~ ('microsoft.datareplication/replicationvaults'))|where not((type =~ ('microsoft.synapse/workspaces/sqlpools')) and ((kind =~ ('v3'))))|where (type !~ ('microsoft.mission/catalogs'))|where (type !~ ('microsoft.mission/communities'))|where (type !~ ('microsoft.mission/communities/communityendpoints'))|where (type !~ ('microsoft.mission/enclaveconnections'))|where (type !~ ('microsoft.mission/virtualenclaves/enclaveendpoints'))|where (type !~ ('microsoft.mission/virtualenclaves/endpoints'))|where (type !~ ('microsoft.mission/externalconnections'))|where (type !~ ('microsoft.mission/internalconnections'))|where (type !~ ('microsoft.mission/communities/transithubs'))|where (type !~ ('microsoft.mission/virtualenclaves'))|where (type !~ ('microsoft.mission/virtualenclaves/workloads'))|where (type !~ ('microsoft.windowspushnotificationservices/registrations'))|where (type !~ ('microsoft.workloads/insights'))|where (type !~ ('microsoft.hanaonazure/sapmonitors'))|where (type !~ ('microsoft.cloudhealth/healthmodels'))|where (type !~ ('microsoft.manufacturingplatform/manufacturingdataservices'))|where (type !~ ('microsoft.windowsesu/multipleactivationkeys'))|where not((type =~ ('microsoft.sql/servers/databases')) and ((kind in~ ('system','v2.0,system','v12.0,system','v12.0,system,serverless','v12.0,user,datawarehouse,gen2,analytics'))))|where not((type =~ ('microsoft.sql/servers')) and ((kind =~ ('v12.0,analytics'))))|where (organizationType contains ('GitHub'))|project name,resourceGroup,locationDisplayName,subscriptionDisplayName,id,type,kind,location,subscriptionId,tags|sort by (tolower(tostring(name))) asc"),
				Facets:           nil,
				ManagementGroups: nil,
				Options:          options,
				Subscriptions:    []*string{&c.SubscriptionId},
			},
			&armresourcegraph.ClientResourcesOptions{},
		)
		if err != nil {
			return nil, fmt.Errorf("cannot list pools: %+v", err)
		}
		if response.Data == nil {
			return resp, nil
		}
		for _, data := range response.Data.([]any) {
			m := data.(map[string]any)
			resp.Data = append(resp.Data, Pool{
				Name:          m["name"].(string),
				ResourceGroup: m["resourceGroup"].(string),
				Location:      m["location"].(string),
				Id:            m["id"].(string),
				Tags:          m["tags"].(map[string]any),
			})
		}
		if len(resp.Data) == int(*response.TotalRecords) {
			break
		}
		options.Skip = p(int32(len(resp.Data)))
	}
	return resp, nil
}

func p[T any](input T) *T {
	return &input
}
