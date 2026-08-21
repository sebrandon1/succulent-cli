package lib

import "net/url"

const (
	StatusUp          = "up"
	StatusDown        = "down"
	StatusError       = "error"
	StatusFailed      = "failed"
	StatusUnreachable = "unreachable"

	NodeTypeInstaller = "installer"
	NodeTypeMaster    = "master"
	NodeTypeWorker    = "worker"
	NodeTypeBootstrap = "bootstrap"

	formFieldOwner     = "owner"
	formFieldMailTo    = "mail_to"
	formFieldMailToAlt = "mailto"

	endpointRoot                 = "/"
	endpointInfoPlan             = "/infoplan/%s"
	endpointLog                  = "/ztp_log/%s"
	endpointReprovision          = "/exposecreate"
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
	Name     string `json:"name"`
	Status   string `json:"status"`
	IP       string `json:"ip,omitempty"`
	NodeType string `json:"node_type,omitempty"`
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

func (r *ReprovisionRequest) FormValues() url.Values {
	data := url.Values{
		"parameter_mail_to":            {r.Email},
		"parameter_owner":              {r.Owner},
		"parameter_tag":                {r.Tag},
		"parameter_version":            {r.Version},
		"parameter_disk_size":          {r.DiskSize},
		"parameter_virtual_workers":    {r.VirtualWorkers},
		"parameter_additional_workers": {r.AdditionalWorkers},
	}

	setIfNotEmpty(data, "parameter_openshift_image", r.OpenshiftImage)
	setIfNotEmpty(data, "parameter_end_date", r.EndDate)
	setIfNotEmpty(data, "additional_params", r.KcliParams)

	return data
}

func (r *SNOProvisionRequest) FormValues() url.Values {
	data := url.Values{
		formFieldOwner:     {r.Owner},
		formFieldMailToAlt: {r.Email},
	}

	setIfNotEmpty(data, "ocp_tag", r.OCPTag)
	setIfNotEmpty(data, "ocp_release_type", r.ReleaseType)
	setIfNotEmpty(data, "full_ocp_tag", r.FullOCPTag)
	setIfNotEmpty(data, "full_image_name", r.FullImageName)

	return data
}

func (r *ZTPRequest) FormValues() url.Values {
	data := url.Values{
		formFieldOwner:  {r.Owner},
		formFieldMailTo: {r.Email},
	}

	setIfNotEmpty(data, "sno_tag", r.SNOTag)
	setIfNotEmpty(data, "sno_release", r.SNORelease)
	setIfNotEmpty(data, "sno_full_tag", r.SNOFullTag)
	setIfNotEmpty(data, "ztp_tag", r.ZTPTag)
	setIfNotEmpty(data, "ztp_release", r.ZTPRelease)
	setIfNotEmpty(data, "ztp_full_tag", r.ZTPFullTag)
	setIfNotEmpty(data, "ztp_type", r.ZTPType)

	if r.StopBeforeDeployment {
		data.Set("stop_before_deployment", "on")
	}

	setIfNotEmpty(data, "vm-masters-count", r.VMMastersCount)
	setIfNotEmpty(data, "bm-masters-hosts", r.BMMastersHosts)
	setIfNotEmpty(data, "bm-workers-hosts", r.BMWorkersHosts)
	setIfNotEmpty(data, "vm-workers-count", r.VMWorkersCount)

	return data
}

func (r *HypershiftRequest) FormValues() url.Values {
	data := url.Values{
		formFieldOwner:  {r.Owner},
		formFieldMailTo: {r.Email},
	}

	setIfNotEmpty(data, "sno_tag", r.SNOTag)
	setIfNotEmpty(data, "sno_release", r.SNORelease)
	setIfNotEmpty(data, "sno_full_tag", r.SNOFullTag)
	setIfNotEmpty(data, "hcp_tag", r.HCPTag)
	setIfNotEmpty(data, "hcp_release", r.HCPRelease)
	setIfNotEmpty(data, "hcp_full_tag", r.HCPFullTag)
	setIfNotEmpty(data, "vm-workers-count", r.VMWorkersCount)
	setIfNotEmpty(data, "image_override", r.ImageOverride)

	return data
}

func setIfNotEmpty(data url.Values, key, value string) {
	if value != "" {
		data.Set(key, value)
	}
}
