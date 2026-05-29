package lib

const (
	StatusUp   = "up"
	StatusDown = "down"

	formFieldOwner     = "owner"
	formFieldMailTo    = "mail_to"
	formFieldMailToAlt = "mailto"

	endpointRoot                 = "/"
	endpointInfoPlan             = "/infoplan/%s"
	endpointLog                  = "/ztp_log/%s"
	endpointReprovision          = "/exposeform/%s"
	endpointDelete               = "/exposedelete"
	endpointSNOProvision         = "/sno/%s"
	endpointSNOKubeconfig        = "/sno_kubeconfig/%s"
	endpointZTPProvision         = "/create_ztp"
	endpointZTPKubeconfig        = "/ztp_kubeconfig"
	endpointHypershiftProvision  = "/create_hypershift"
	endpointHypershiftKubeconfig = "/hypershift_kubeconfig"
)

// NodeInfo represents a single VM/node from the infoplan page.
type NodeInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	IP     string `json:"ip,omitempty"`
}

// ClusterInfo represents the parsed result of the infoplan page.
type ClusterInfo struct {
	Environment  string     `json:"environment"`
	PlanName     string     `json:"plan_name,omitempty"`
	Client       string     `json:"client,omitempty"`
	CreationDate string     `json:"creation_date,omitempty"`
	InstallerIP  string     `json:"installer_ip,omitempty"`
	Nodes        []NodeInfo `json:"nodes"`
}

// EnvironmentInfo represents an environment from the main page.
type EnvironmentInfo struct {
	Name  string `json:"name"`
	Group string `json:"group,omitempty"`
}

// EnvironmentDetail represents an environment with enriched info from the infoplan page.
type EnvironmentDetail struct {
	Name         string `json:"name"`
	Group        string `json:"group,omitempty"`
	Owner        string `json:"owner,omitempty"`
	CreationDate string `json:"creation_date,omitempty"`
	NodeCount    int    `json:"node_count"`
	NodesUp      int    `json:"nodes_up"`
	InstallerIP  string `json:"installer_ip,omitempty"`
	Status       string `json:"status"`
}

// HypershiftRequest represents the Hypershift provisioning form data.
type HypershiftRequest struct {
	Owner          string `json:"owner"`
	Email          string `json:"mail_to"`
	SNOTag         string `json:"sno_tag,omitempty"`
	SNORelease     string `json:"sno_release,omitempty"`
	SNOFullTag     string `json:"sno_full_tag,omitempty"`
	HCPTag         string `json:"hcp_tag,omitempty"`
	HCPRelease     string `json:"hcp_release,omitempty"`
	HCPFullTag     string `json:"hcp_full_tag,omitempty"`
	VMWorkersCount string `json:"vm_workers_count,omitempty"`
	ImageOverride  string `json:"image_override,omitempty"`
}

// ZTPRequest represents the ZTP provisioning form data.
type ZTPRequest struct {
	Owner                string `json:"owner"`
	Email                string `json:"mail_to"`
	SNOTag               string `json:"sno_tag,omitempty"`
	SNORelease           string `json:"sno_release,omitempty"`
	SNOFullTag           string `json:"sno_full_tag,omitempty"`
	ZTPTag               string `json:"ztp_tag,omitempty"`
	ZTPRelease           string `json:"ztp_release,omitempty"`
	ZTPFullTag           string `json:"ztp_full_tag,omitempty"`
	ZTPType              string `json:"ztp_type,omitempty"`
	StopBeforeDeployment bool   `json:"stop_before_deployment,omitempty"`
	VMMastersCount       string `json:"vm_masters_count,omitempty"`
	BMMastersHosts       string `json:"bm_masters_hosts,omitempty"`
	BMWorkersHosts       string `json:"bm_workers_hosts,omitempty"`
	VMWorkersCount       string `json:"vm_workers_count,omitempty"`
}

// ReprovisionRequest represents the MNO reprovision form data.
type ReprovisionRequest struct {
	Email             string `json:"email"`
	Owner             string `json:"owner"`
	Tag               string `json:"tag"`
	Version           string `json:"version"`
	OpenshiftImage    string `json:"openshift_image,omitempty"`
	DiskSize          string `json:"disk_size,omitempty"`
	VirtualWorkers    string `json:"virtual_workers,omitempty"`
	AdditionalWorkers string `json:"additional_workers,omitempty"`
	EndDate           string `json:"end_date,omitempty"`
	KcliParams        string `json:"kcli_params,omitempty"`
}

// SNOProvisionRequest represents the SNO provision form data.
type SNOProvisionRequest struct {
	Owner         string `json:"owner"`
	Email         string `json:"mail_to"`
	OCPTag        string `json:"ocp_tag,omitempty"`
	ReleaseType   string `json:"ocp_release_type,omitempty"`
	FullOCPTag    string `json:"full_ocp_tag,omitempty"`
	FullImageName string `json:"full_image_name,omitempty"`
}
