package cmd

const (
	cmdNameConfig           = "config"
	cmdNameInfo             = "info"
	cmdNameKubeconfig       = "kubeconfig"
	cmdNameProvision        = "provision"
	defaultRemoteUser       = "root"
	defaultRemotePath       = "/root/ocp/auth/kubeconfig"
	defaultDestDir          = "Downloads"
	defaultMaxWaitMinutes   = 60
	defaultPollIntervalSecs = 30
	minPollIntervalSecs     = 5
)
