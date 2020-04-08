package kubernikus

import (
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"testing"
)

func TestNodeGroupTargetSize(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		nodePoolSize := 4

		client := &doClientMock{}
		ng := testNodeGroup(client, &models.NodePool{
			Name: "test",
			Size: int64(nodePoolSize),
		})

		size, err := ng.TargetSize()
		assert.NoError(t, err, "there should be no error getting the target size from the node group")
		assert.Equal(t, nodePoolSize, size, "target size should be equal")
	})
}

type doClientMock struct {
	mock.Mock
}

func (m *doClientMock) ListNodePools(clusterName string) ([]models.NodePool, error) {
	args := m.Called(clusterName)
	return args.Get(0).([]models.NodePool), args.Error(1)
}

func (m *doClientMock) UpdateNodePool(clusterName string, nodePoolName string, opts UpdateNodePoolOpts) (models.NodePool, error) {
	args := m.Called(clusterName, nodePoolName, opts)
	return args.Get(0).(models.NodePool), args.Error(1)
}

func (m *doClientMock) DeleteNode(clusterName string, nodePoolName string, nodeName string) (models.NodePool, error) {
	return models.NodePool{}, cloudprovider.ErrNotImplemented
}

func testNodeGroup(client nodeGroupClient, nodePool *models.NodePool) *kubernikusNodeGroup {
	return &kubernikusNodeGroup{
		client: client,
		clusterID: "1",
		nodePool: nodePool,
	}
}