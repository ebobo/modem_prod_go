package model

// Define the modem struct to represent the modem data
type Modem struct {
	MacAddress  string `json:"mac_address" db:"mac_address"`
	IPV6        string `json:"ipv6" db:"ipv6"`
	SwitchPort  int    `json:"switch_port" db:"switch_port"`
	Model       string `json:"model" db:"model"`
	State       int    `json:"state" db:"state"` //0: unknown, 1: normal, 2: busy, 3: error
	Firmware    string `json:"firmware" db:"firmware"`
	Serial      string `json:"serial" db:"serial"`
	Kernel      string `json:"kernel" db:"kernel"`
	Upgraded    bool   `json:"upgraded" db:"upgraded"`
	LastUpdated int    `json:"last_updated" db:"last_updated"`
	FailCount   int    `json:"fail_count" db:"fail_count"`
	SIMProvider string `json:"sim_provider" db:"sim_provider"`
	SIMStatus   bool   `json:"sim_status" db:"sim_status"`
	IMEI        string `json:"imei" db:"imei"`
	ICCID       string `json:"iccid" db:"iccid"`
	IMSI        string `json:"imsi" db:"imsi"`
	Progress    int    `json:"progress" db:"progress"`
}
