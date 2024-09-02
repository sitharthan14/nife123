package duplo

type FaultByTenant struct {
	TenantId     string `json:"tenantId"`
	ModuleName   string `json:"moduleName"`
	ResourceName string `json:"resourceName"`
	Description  string `json:"description"`
}

type UpdateLBConfig struct {
	ReplicationControllerName string
	LBType                    int
	Protocol                  string
	Port                      string
	ExternalPort              int
	State                     string
	IsInternal                bool
	HealthCheckUrl            string
	CertificateArn            string
}

type CreateService struct {
	Replicas          int
	Cloud             int
	AgentPlatform     int
	Name              string
	Volumes           string
	AllocationTags    string
	DockerImage       string
	TenantId          string
	NetworkId         string
	OtherDockerConfig string
	ExtraConfig       string
}

type LoadBalancer struct {
	HealthCheckConfig         string
	LbType                    int
	Port                      int
	ExternalPort              int
	IsInternal                bool
	IsNative                  bool
	HealthCheckUrl            string
	Protocol                  string
	CertificateArn            string
	ReplicationControllerName string
}

type DuploDetails struct {
	AppName             string
	Image               string
	UserId              string
	Tag                 string
	EnvArgs             string
	AgentPlatForm       int
	InternalPort        int
	ExternalPort        int
	ExternalBaseAddress string
}

type UpdateDuplo struct {
	AppName             string
	Image               string
	Status              string
	UserId              string
	InternalPort        int
	ExternalPort        int
	AgentPlatForm       int
	ExternalBaseAddress string
}

type CreateSecret struct {
	SecretName string
	SecretType string
	SecretData map[string]interface{}
}

// type SecretData struct {
//     dockerconfigjson Dockerconfig  `json:".dockerconfigjson"`
// }

type Dockerconfig struct {
	auths map[string]interface{}
}

type SecretCredential struct {
	username string
	password string
	email    string
	auth     string
}

type DeleteSecret struct {
	SecretName          string
	ExternalBaseAddress string
	TenantId            string
}

type Getsecret struct {
	Name                string
	TenantId            string
	ExternalBaseAddress string
}
