package device

type T_IOInterface struct {
	InterfaceName string `json:"interfaceName"`
	InterfaceType string `json:"interfaceType"`
	InterfaceIP   string `json:"interfaceIP"`
	InterfacePort int    `json:"interfacePort"`
}