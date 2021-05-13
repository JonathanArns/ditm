package plugin

import "github.com/docker/go-plugins-helpers/network"

func Test(a network.Driver) {
	//
}

type Driver struct {
}

func (d *Driver) GetCapabilities() (*network.CapabilitiesResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) AllocateNetwork(r *network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) FreeNetwork(r *network.FreeNetworkRequest) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) CreateEndpoint(r *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) EndpointInfo(r *network.InfoRequest) (*network.InfoResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) Join(r *network.JoinRequest) (*network.JoinResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) Leave(r *network.LeaveRequest) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) DiscoverNew(n *network.DiscoveryNotification) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) DiscoverDelete(n *network.DiscoveryNotification) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) ProgramExternalConnectivity(r *network.ProgramExternalConnectivityRequest) error {
	panic("not implemented") // TODO: Implement
}

func (d *Driver) RevokeExternalConnectivity(r *network.RevokeExternalConnectivityRequest) error {
	panic("not implemented") // TODO: Implement
}
