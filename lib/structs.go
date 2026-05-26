package lib

const (
	StatusUp   = "up"
	StatusDown = "down"
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
