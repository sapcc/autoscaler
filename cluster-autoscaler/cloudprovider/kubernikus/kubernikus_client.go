package kubernikus

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/pkg/errors"
	kubernikuscli "github.com/sapcc/kubernikus/pkg/api/client"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
)

const (
	pollInterval = 5 * time.Second
	timeout = 5 * time.Minute
)

type (
	nodeGroupClient interface {
		// ListNodePools returns a list of node pool in the given cluster or an error.
		ListNodePools(clusterName string) ([]models.NodePool, error)

		// UpdateNodePools updates the node pool in the given cluster and returns the updated version or an error.
		UpdateNodePool(clusterName string, nodePoolName string, opts UpdateNodePoolOpts) (models.NodePool, error)

		// DeleteNode terminates the node in the given cluster, node pool and returns the updated node pool or an error.
		DeleteNode(clusterName string, nodePoolName string, nodeName string) (models.NodePool, error)

		// GetAvailableMachineTypes returns the list of available openstack flavors.
		GetAvailableMachineTypes() ([]string, error)
	}

	kubernikusClient struct {
		openstackClient *openstackClient
		client          *kubernikuscli.Kubernikus
	}

	openstackClient struct {
		*tokens.AuthOptions
		provider  *gophercloud.ProviderClient
		computeV2 *gophercloud.ServiceClient
	}

	UpdateNodePoolOpts struct {
		TargetSize int
	}
)

func newKubernikusClient(cfg Config) (*kubernikusClient, error) {
	kURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing url %s", cfg.URL)
	}

	transport := kubernikuscli.DefaultTransportConfig().
		WithSchemes([]string{kURL.Scheme}).
		WithHost(kURL.Host).
		WithBasePath(kURL.EscapedPath())

	os, err := newOpenstackClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error creating openstack client")
	}

	if err = os.authenticate(); err != nil {
		return nil, errors.Wrap(err, "error during openstack authentication")
	}

	return &kubernikusClient{
		openstackClient: os,
		client:          kubernikuscli.NewHTTPClientWithConfig(nil, transport),
	}, nil
}

func (k *kubernikusClient) authFunc() runtime.ClientAuthInfoWriterFunc {
	return runtime.ClientAuthInfoWriterFunc(
		func(req runtime.ClientRequest, reg strfmt.Registry) error {
			req.SetHeaderParam("X-AUTH-TOKEN", k.openstackClient.TokenID)
			return nil
		})
}

func (k *kubernikusClient) ListNodePools(clusterName string) ([]models.NodePool, error) {
	ok, err := k.showCluster(clusterName)
	if err != nil {
		return nil, err
	}
	return ok.Spec.NodePools, nil
}

func (k *kubernikusClient) UpdateNodePool(clusterName string, nodePoolName string, opts UpdateNodePoolOpts) (models.NodePool, error) {
	kluster, err := k.showCluster(clusterName)
	if err != nil {
		return models.NodePool{}, err
	}

	nodePools := make([]models.NodePool, len(kluster.Spec.NodePools))
	for idx, np := range kluster.Spec.NodePools {
		nodePools[idx] = updateNodePool(np, nodePoolName, opts)
	}
	kluster.Spec.NodePools = nodePools

	updatedKluster, err := k.updateCluster(kluster)
	if err != nil {
		return models.NodePool{}, err
	}

	return filterNodePools(updatedKluster.Spec.NodePools, nodePoolName)
}

func (k *kubernikusClient) DeleteNode(clusterName string, nodePoolName string, nodeName string) (models.NodePool, error) {
	if err := k.openstackClient.DeleteServer(nodeName); err != nil {
		return models.NodePool{}, err
	}

	kluster, err := k.showCluster(clusterName)
	if err != nil {
		return models.NodePool{}, err
	}

	return filterNodePools(kluster.Spec.NodePools, nodePoolName)
}

func (k *kubernikusClient) GetAvailableMachineTypes() ([]string, error) {
	meta, err := k.getOpenstackMetadata()
	if err != nil {
		return nil, err
	}

	machTypes := make([]string, len(meta.Flavors))
	for idx, t := range meta.Flavors {
		machTypes[idx] = t.Name
	}
	return machTypes, nil
}

func (k *kubernikusClient) getOpenstackMetadata() (*models.OpenstackMetadata, error) {
	ok, err := k.client.Operations.GetOpenstackMetadata(
		operations.NewGetOpenstackMetadataParams(), k.authFunc(),
	)

	switch err.(type) {
	case *operations.GetOpenstackMetadataDefault:
		result := err.(*operations.GetOpenstackMetadataDefault)
		return nil, errors.Errorf("error getting openstack metadata %s", *result.Payload.Message)
	case error:
		return nil, errors.Wrapf(err, "error getting openstack metadata")
	}
	return ok.Payload, nil

}

func (k *kubernikusClient) showCluster(clusterName string) (*models.Kluster, error) {
	ok, err := k.client.Operations.ShowCluster(
		operations.NewShowClusterParams().WithName(clusterName), k.authFunc(),
	)

	switch err.(type) {
	case *operations.ShowClusterDefault:
		result := err.(*operations.ShowClusterDefault)
		return nil, errors.Errorf("error getting cluster %s: %s", clusterName, *result.Payload.Message)
	case error:
		return nil, errors.Wrapf(err, "error getting cluster: %s", clusterName)
	}
	return ok.Payload, nil
}

func (k *kubernikusClient) updateCluster(cluster *models.Kluster) (*models.Kluster, error) {
	newCluster, err := k.client.Operations.UpdateCluster(
		operations.NewUpdateClusterParams().WithName(cluster.Name).WithBody(cluster), k.authFunc(),
	)

	switch err.(type) {
	case *operations.UpdateClusterDefault:
		result := err.(*operations.UpdateClusterDefault)
		return nil, errors.Errorf(*result.Payload.Message)
	case error:
		return nil, errors.Wrap(err, "Error updating cluster")
	}

	return newCluster.Payload, err
}

func filterNodePools(nodePools []models.NodePool, nodePoolName string) (models.NodePool, error) {
	for _, np := range nodePools {
		if np.Name == nodePoolName {
			return np, nil
		}
	}

	return models.NodePool{}, fmt.Errorf("no nodepool with name %s found", nodePoolName)
}

func updateNodePool(nodePool models.NodePool, nodePoolName string, opts UpdateNodePoolOpts) models.NodePool {
	// That's not the nodePool you're looking for.
	if nodePool.Name != nodePoolName {
		return nodePool
	}

	newNodePool := nodePool.DeepCopy()

	if opts.TargetSize != 0 {
		newNodePool.Size = int64(opts.TargetSize)
	}

	return *newNodePool
}

func newOpenstackClient(cfg Config) (*openstackClient, error) {
	authOpts := &tokens.AuthOptions{
		IdentityEndpoint:            cfg.IdentityEndpoint,
		Username:                    cfg.Username,
		UserID:                      cfg.UserID,
		Password:                    cfg.Password,
		DomainID:                    cfg.DomainID,
		DomainName:                  cfg.DomainName,
		AllowReauth:                 true,
		ApplicationCredentialID:     cfg.ApplicationCredentialID,
		ApplicationCredentialName:   cfg.ApplicationCredentialName,
		ApplicationCredentialSecret: cfg.ApplicationCredentialSecret,
	}

	if cfg.ProjectID != "" || cfg.ProjectName != "" || cfg.ProjectDomainID != "" || cfg.ProjectDomainName != "" {
		authOpts.Scope = tokens.Scope{
			ProjectID:   cfg.ProjectID,
			ProjectName: cfg.ProjectName,
			DomainID:    cfg.ProjectDomainID,
			DomainName:  cfg.ProjectDomainName,
		}
	}

	provider, err := openstack.NewClient(cfg.IdentityEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "error creating gophercloud provider client")
	}

	os := &openstackClient{
		AuthOptions: authOpts,
		provider:    provider,
		computeV2:   nil,
	}

	if err := os.authenticate(); err != nil {
		return nil, err
	}

	os.computeV2, err = openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	return os, err
}

func (o *openstackClient) authenticate() error {
	if o.provider.TokenID != "" {
		o.TokenID = o.provider.TokenID
	}

	return openstack.AuthenticateV3(o.provider, o, gophercloud.EndpointOpts{})
}

func (o *openstackClient) DeleteServer(nodeName string) error {
	listOpts := servers.ListOpts{
		Name: nodeName,
	}

	pager, err := servers.List(o.computeV2, listOpts).AllPages()
	if err != nil {
		return err
	}

	serverList, err := servers.ExtractServers(pager)
	if err != nil {
		return err
	}

	nrServers := len(serverList)
	if nrServers == 0 {
		return fmt.Errorf("no a single server found with name: %s", nodeName)
	} else if nrServers > 1 {
		return fmt.Errorf("multiple servers found with name: %s", nodeName)
	}

	err = servers.Delete(o.computeV2, serverList[0].ID).ExtractErr()
	return err
}
