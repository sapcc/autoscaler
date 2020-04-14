package kubernikus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	scalererrors "k8s.io/autoscaler/cluster-autoscaler/utils/errors"
	"k8s.io/klog"
)

//TODO
// GPULabel is the label added to nodes with GPU resource.
const (

	// GPULabel is the label added to nodes with GPU resource.
	GPULabel = ""

	// Label identifying the Kubernikus nodepool.
	nodePoolLabel = "ccloud.sap.com/nodepool"

	scaleToZeroSupported = false
)

var availableGPUTypes = map[string]struct{}{
	"nvidia-tesla-v100": {},
}

type kubernikusCloudProvider struct {
	kubernikusManager *kubernikusManager
	resourceLimiter   *cloudprovider.ResourceLimiter
}

func newKubernikusCloudProvider(kubernikusManager *kubernikusManager, resourceLimiter *cloudprovider.ResourceLimiter) (cloudprovider.CloudProvider, error) {
	if err := kubernikusManager.Refresh(); err != nil {
		return nil, errors.Wrap(err, "refreshing node pools failed")
	}

	return &kubernikusCloudProvider{
		kubernikusManager: kubernikusManager,
		resourceLimiter:   resourceLimiter,
	}, nil
}

func (kcp *kubernikusCloudProvider) Name() string {
	return cloudprovider.KubernikusProviderName
}

func (kcp *kubernikusCloudProvider) NodeGroups() []cloudprovider.NodeGroup {
	groups := make([]cloudprovider.NodeGroup, len(kcp.kubernikusManager.nodeGroups))
	for i, g := range kcp.kubernikusManager.nodeGroups {
		groups[i] = g
	}
	return groups
}

func (kcp *kubernikusCloudProvider) NodeGroupForNode(node *apiv1.Node) (cloudprovider.NodeGroup, error) {
	lbls := node.GetLabels()
	if lbls == nil {
		return nil, fmt.Errorf("error identifying nodeGroup. no labels present on node: %s", node.GetName())
	}

	nodePoolName, ok := lbls[nodePoolLabel]
	if !ok {
		return nil, fmt.Errorf("error identifying nodeGroup. label %s is not present on node %s", nodePoolLabel, node.GetName())
	}

	for _, np := range kcp.kubernikusManager.nodeGroups {
		if nodePoolName == np.Id() {
			return np, nil
		}
	}

	return &kubernikusNodeGroup{}, fmt.Errorf("error identifying nodeGroup. no nodeGroup exists for node %s, nodeGroup %s", node.GetName(), nodePoolName)
}

func (kcp *kubernikusCloudProvider) Pricing() (cloudprovider.PricingModel, scalererrors.AutoscalerError) {
	return nil, cloudprovider.ErrNotImplemented
}

func (kcp *kubernikusCloudProvider) GetAvailableMachineTypes() ([]string, error) {
	return kcp.kubernikusManager.client.GetAvailableMachineTypes()
}

func (kcp *kubernikusCloudProvider) NewNodeGroup(machineType string, labels map[string]string, systemLabels map[string]string, taints []apiv1.Taint, extraResources map[string]resource.Quantity) (cloudprovider.NodeGroup, error) {
	// TODO: Creation of a new NodeGroup.
	return nil, cloudprovider.ErrNotImplemented
}

func (kcp *kubernikusCloudProvider) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return kcp.resourceLimiter, nil
}

func (kcp *kubernikusCloudProvider) GPULabel() string {
	return GPULabel
}

func (kcp *kubernikusCloudProvider) GetAvailableGPUTypes() map[string]struct{} {
	return availableGPUTypes
}

func (kcp *kubernikusCloudProvider) Cleanup() error {
	return nil
}

func (kcp *kubernikusCloudProvider) Refresh() error {
	return kcp.kubernikusManager.Refresh()
}

func BuildKubernikus(opts config.AutoscalingOptions, nodeGroupDiscoveryOpts cloudprovider.NodeGroupDiscoveryOptions, resourceLimiter *cloudprovider.ResourceLimiter) cloudprovider.CloudProvider {
	cfg := Config{}

	if opts.CloudConfig != "" {
		configFile, err := os.Open(opts.CloudConfig)
		if err != nil {
			klog.Fatalf("Couldn't open cloud provider configuration %s: %#v", opts.CloudConfig, err)
		}
		defer configFile.Close()

		body, err := ioutil.ReadAll(configFile)
		if err != nil {
			klog.Fatalf("Couldn't read cloud provider configuration %s: %#v", opts.CloudConfig, err)
		}

		if err := json.Unmarshal(body, &cfg); err != nil {
			klog.Fatalf("Couldn't unmarshal cloud provider configuration %s: %#v", opts.CloudConfig, err)
		}
	}

	if err := cfg.validate(); err != nil {
		klog.Fatalf("Configuration of kubernikus cloud provider invalid: %v", err)
	}

	manager, err := newKubernikusManager(cfg)
	if err != nil {
		klog.Fatalf("Failed to create kubernikus manager: %v", err)
	}

	provider, err := newKubernikusCloudProvider(manager, resourceLimiter)
	if err != nil {
		klog.Fatalf("Failed to create kubernikus cloud provider: %v", err)
	}

	return provider
}
