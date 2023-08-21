package pkg

type WinSCPProfile struct {
	ProfileName       string `json:"winscp_profile"`
	Hostname          string `json:"winscp_hostname"`
	Username          string `json:"winscp_username"`
	Password          string `json:"winscp_clear_password,omitempty"`
	Port              string `json:"winscp_port,omitempty"`
	Protocol          string `json:"winscp_protocol,omitempty"`
	ProxyServer       string `json:"winscp_proxyserver,omitempty"`
	ProxyPort         string `json:"winscp_proxyport,omitempty"`
	PublicKeyFilePath string `json:"winscp_ppkfilepath,omitempty"`
	PublicKeyRAWText  string `json:"winscp_ppk_rawdata,omitempty"`
	HostFingerPrint   string `json:"winscp_hostfingerprint,omitempty"`
	RemotePath        string `json:"winscp_remote_path,omitempty"`
	LocalPath         string `json:"winscp_local_path,omitempty"`
	//HostPublicKey     string `json:"winscp_hostkey,omitempty"`
}

type ConnectionTestResult struct {
	ProfileName    string `json:"winscpprofilename"`
	Hostname       string `json:"hostname"`
	Port           string `json:"port"`
	Username       string `json:"username"`
	AuthMethod     string `json:"authmethod"`
	ConnTestResult string `json:"conntestresult"`
	FurtherInfo    string `json:"furtherinfo"`
}
