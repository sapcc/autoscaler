package kubernikus

import (
	"fmt"
	"net/url"

	"github.com/sapcc/kubernikus/pkg/api/models"
	kubernikuscli "github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/klog"
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
		client *kubernikuscli.KubernikusClient
	}

	UpdateNodePoolOpts struct {
		TargetSize int
	}
)

func newKubernikusClient(cfg Config) (*kubernikusClient, error) {
	kURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}

	os := kubernikuscli.NewOpenstackClient()
	os.AllowReauth = true
	os.IdentityEndpoint = cfg.IdentityEndpoint
	os.UserID = cfg.UserID
	os.Username = cfg.Username
	os.Password = cfg.Password
	os.DomainID = cfg.UserDomainID
	os.DomainName = cfg.UserDomainName
	os.DomainID = cfg.DomainID
	os.DomainName = cfg.DomainName
	os.ApplicationCredentialID = cfg.ApplicationCredentialID
	os.ApplicationCredentialName = cfg.ApplicationCredentialName
	os.ApplicationCredentialSecret = cfg.ApplicationCredentialSecret
	os.Scope.ProjectID = cfg.ProjectID
	os.Scope.ProjectName = cfg.ProjectName
	os.Scope.DomainID = cfg.ProjectDomainID
	os.Scope.DomainName = cfg.ProjectDomainName

	if err := os.Setup(); err != nil {
		klog.Fatal("failed to setup openstack provider: %v", err)
	}

	if err := os.Authenticate(); err != nil {
		klog.Fatal("failed to authenticate with given credentials: %v", err)
	}

	return &kubernikusClient{
		client: kubernikuscli.NewKubernikusClient(kURL, os.TokenID),
	}, nil
}

func (k *kubernikusClient) ListNodePools(clusterName string) ([]models.NodePool, error) {
	return k.client.ListNodePools(clusterName)
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
	return k.client.ShowCluster(clusterName)
}

func (k *kubernikusClient) updateCluster(cluster *models.Kluster) error {
	return k.client.UpdateCluster(cluster)
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
