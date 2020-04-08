package kubernikus

import (
	"errors"
	"os"
)

type (
	// Config is the configuration of the kubernikus cloud provider.
	Config struct {
		ClusterName string `json:"cluster_name"`
		URL string `json:"url"`

		IdentityEndpoint string `json:"-"`
		UserID  string `json:"-"`
		Username  string `json:"-"`
		Password  string `json:"-"`
		UserDomainID  string `json:"-"`
		UserDomainName string `json:"-"`
		ProjectID  string `json:"-"`
		ProjectName  string `json:"-"`
		DomainID  string `json:"-"`
		DomainName  string `json:"-"`
		ProjectDomainID  string `json:"-"`
		ProjectDomainName  string `json:"-"`
		ApplicationCredentialID string `json:"-"`
		ApplicationCredentialName string `json:"-"`
		ApplicationCredentialSecret string `json:"-"`
	}

	kubernikusManager struct {
		client  nodeGroupClient
		nodeGroups       []*kubernikusNodeGroup
		clusterName,
		token string
	}
)

func (c *Config) fromEnv() {
	c.ClusterName = os.Getenv("CLUSTER_NAME")
	c.URL = os.Getenv("KUBERNIKUS_URL")

	c.IdentityEndpoint = os.Getenv("OS_AUTH_URL")
	c.UserID = os.Getenv("OS_USER_ID")
	c.Username = os.Getenv("OS_USERNAME")
	c.Password = os.Getenv("OS_PASSWORD")
	c.UserDomainID = os.Getenv("OS_USER_DOMAIN_ID")
	c.UserDomainName = os.Getenv("OS_USER_DOMAIN_NAME")
	c.ProjectID = os.Getenv("OS_PROJECT_ID")
	c.ProjectName = os.Getenv("OS_PROJECT_NAME")
	c.DomainID = os.Getenv("OS_DOMAIN_ID")
	c.DomainName = os.Getenv("OS_DOMAIN_NAME")
	c.ProjectDomainID = os.Getenv("OS_PROJECT_DOMAIN_ID")
	c.ProjectDomainName = os.Getenv("OS_PROJECT_DOMAIN_NAME")
	c.ApplicationCredentialID = os.Getenv("OS_APPLICATION_CREDENTIAL_ID")
	c.ApplicationCredentialName = os.Getenv("OS_APPLICATION_CREDENTIAL_NAME")
	c.ApplicationCredentialSecret = os.Getenv("OS_APPLICATION_CREDENTIAL_SECRET")
}

func (c *Config) validate() error {
	if c.ClusterName == "" {
		return errors.New("cluster_name not provided. aborting")
	}

	if c.URL == "" {
		return errors.New("url not provided. aborting")
	}

	return nil
}

func newKubernikusManager(cfg Config) (*kubernikusManager, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	kCli, err := newKubernikusClient(cfg)
	if err != nil {
		return nil, err
	}

	return &kubernikusManager{
		client: kCli,
		nodeGroups:       make([]*kubernikusNodeGroup, 0),
		clusterName:      cfg.ClusterName,
	}, nil
}

func (km *kubernikusManager) Refresh() error {
	nodePools, err := km.client.ListNodePools(km.clusterName)
	if err != nil {
		return err
	}

	nodeGroups := make([]*kubernikusNodeGroup, len(nodePools))
	for idx, np := range nodePools {
		nodeGroups[idx] = &kubernikusNodeGroup{
			client: km.client,
			clusterID: km.clusterName,
			nodePool: &np,
		}
	}

	km.nodeGroups = nodeGroups
	return nil
}
