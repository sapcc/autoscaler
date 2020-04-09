package kubernikus

import (
	"fmt"
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/pkg/errors"
	kubernikuscli "github.com/sapcc/kubernikus/pkg/api/client"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
)

type (
	nodeGroupClient interface {
		// ListNodePools returns a list of node pool in the given cluster or an error.
		ListNodePools(clusterName string) ([]models.NodePool, error)

		// UpdateNodePools updates the node pool in the given cluster and returns the updated version or an error.
		UpdateNodePool(clusterName string, nodePoolName string, opts UpdateNodePoolOpts) (models.NodePool, error)

		// DeleteNode terminates the node in the given cluster, node pool and returns the updated node pool or an error.
		DeleteNode(clusterName string, nodePoolName string, nodeName string) (models.NodePool, error)
	}

	kubernikusClient struct {
		token  string
		client *kubernikuscli.Kubernikus
	}

	openstackClient struct {
		*tokens.AuthOptions
		provider *gophercloud.ProviderClient
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
		token:  os.provider.TokenID,
		client: kubernikuscli.NewHTTPClientWithConfig(nil, transport),
	}, nil
}

func (k *kubernikusClient) authFunc() runtime.ClientAuthInfoWriterFunc {
	return runtime.ClientAuthInfoWriterFunc(
		func(req runtime.ClientRequest, reg strfmt.Registry) error {
			req.SetHeaderParam("X-AUTH-TOKEN", k.token)
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

	cp := kluster.DeepCopy()
	for idx, np := range cp.Spec.NodePools {
		if np.Name == nodePoolName {
			cp.Spec.NodePools[idx] = updateNodePool(np, opts)
		}
	}

	if err := k.updateCluster(cp); err != nil {
		return models.NodePool{}, err
	}

	updatedKluster, err := k.showCluster(clusterName)
	if err != nil {
		return models.NodePool{}, err
	}

	return filterNodePools(updatedKluster.Spec.NodePools, nodePoolName)
}

func (k *kubernikusClient) DeleteNode(clusterName string, nodePoolName string, nodeName string) (models.NodePool, error) {
	return models.NodePool{}, cloudprovider.ErrNotImplemented
}

func (k *kubernikusClient) showCluster(clusterName string) (*models.Kluster, error) {
	ok, err := k.client.Operations.ShowCluster(
		operations.NewShowClusterParams().WithName(clusterName),
		k.authFunc(),
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

func (k *kubernikusClient) updateCluster(cluster *models.Kluster) error {
	_, err := k.client.Operations.UpdateCluster(
		operations.NewUpdateClusterParams().WithBody(cluster),
		k.authFunc(),
	)

	switch err.(type) {
	case *operations.UpdateClusterDefault:
		result := err.(*operations.UpdateClusterDefault)
		return errors.Errorf(*result.Payload.Message)
	case error:
		return errors.Wrap(err, "Error updating cluster")
	}
	return nil
}

func filterNodePools(nodePools []models.NodePool, nodePoolName string) (models.NodePool, error) {
	for _, np := range nodePools {
		if np.Name == nodePoolName {
			return np, nil
		}
	}

	return models.NodePool{}, fmt.Errorf("no nodepool with name %s found", nodePoolName)
}

func updateNodePool(nodePool models.NodePool, opts UpdateNodePoolOpts) models.NodePool {
	if opts.TargetSize != 0 {
		nodePool.Size = int64(opts.TargetSize)
	}

	return nodePool
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

	return &openstackClient{
		AuthOptions: authOpts,
		provider:    provider,
	}, nil
}

func (o *openstackClient) authenticate() error {
	if o.provider.TokenID != "" {
		o.TokenID = o.provider.TokenID
	}

	return openstack.AuthenticateV3(o.provider, o, gophercloud.EndpointOpts{})
}
