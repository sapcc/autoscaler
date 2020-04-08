package kubernikus

import (
	"errors"
	"fmt"

	"github.com/sapcc/kubernikus/pkg/api/models"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)

type kubernikusNodeGroup struct {
	client nodeGroupClient
	clusterID string
	nodePool  *models.NodePool
}

func (kng *kubernikusNodeGroup) MaxSize() int {
	// Currently hardcoded in Kubernikus.
	return 127
}

func (kng *kubernikusNodeGroup) MinSize() int {
	// Currently hardcoded in Kubernikus.
	return 0
}

func (kng *kubernikusNodeGroup) TargetSize() (int, error) {
	return int(kng.nodePool.Size), nil
}

func (kng *kubernikusNodeGroup) IncreaseSize(delta int) error {
	if delta <= 0 {
		return errors.New("delta must be > 0")
	}

	targetSize := int(kng.nodePool.Size) + delta

	if targetSize > kng.MaxSize() {
		return fmt.Errorf(
			"nodepool size would exceed configured maximum. current: %d, desired: %d, maximum: %d",
			kng.nodePool.Size, targetSize, kng.MaxSize(),
		)
	}

	_, err := kng.client.UpdateNodePool(kng.clusterID, kng.nodePool.Name, UpdateNodePoolOpts{TargetSize: targetSize})
	return err
}

func (kng *kubernikusNodeGroup) DeleteNodes([]*apiv1.Node) error {
	return cloudprovider.ErrNotImplemented
}

func (kng *kubernikusNodeGroup) DecreaseTargetSize(delta int) error {
	if delta >= 0 {
		return fmt.Errorf("delta must be >= 0")
	}

	targetSize := int(kng.nodePool.Size) + delta
	if targetSize <= kng.MinSize() {
		return fmt.Errorf(
			"nodepool size would exceed configured minumum. current: %d, desired: %d, minumium: %d",
			kng.nodePool.Size, targetSize, kng.MinSize(),
		)
	}

	_, err := kng.client.UpdateNodePool(kng.clusterID, kng.nodePool.Name, UpdateNodePoolOpts{TargetSize: targetSize})
	return err
}

func (kng *kubernikusNodeGroup) Id() string {
	return kng.nodePool.Name
}

func (kng *kubernikusNodeGroup) Debug() string {
	return fmt.Sprintf("cluster ID: %s (min:%d max:%d)", kng.Id(), kng.MinSize(), kng.MaxSize())
}

func (kng *kubernikusNodeGroup) Nodes() ([]cloudprovider.Instance, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (kng *kubernikusNodeGroup) TemplateNodeInfo() (*schedulernodeinfo.NodeInfo, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (kng *kubernikusNodeGroup) Exist() bool {
	return kng.nodePool != nil
}

func (kng *kubernikusNodeGroup) Create() (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (kng *kubernikusNodeGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (kng *kubernikusNodeGroup) Autoprovisioned() bool {
	return false
}
