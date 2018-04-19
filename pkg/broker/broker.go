package broker

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/osbkit/minibroker/pkg/minibroker"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/osb-broker-lib/pkg/broker"
)

// NewBroker is a hook that is called with the Options the program is run
// with. NewBroker is the place where you will initialize your
// Broker the parameters passed in.
func NewBroker(o Options) (*Broker, error) {
	mb := minibroker.NewClient("")
	err := mb.Init()
	if err != nil {
		return nil, err
	}

	// For example, if your Broker requires a parameter from the command
	// line, you would unpack it from the Options and set it on the
	// Broker here.
	return &Broker{
		Client: mb,
		async:  false,
	}, nil
}

// Broker provides an implementation of broker.Interface
type Broker struct {
	Client *minibroker.Client

	// Indiciates if the broker should handle the requests asynchronously.
	async bool
	// Synchronize go routines.
	sync.RWMutex
}

var _ broker.Interface = &Broker{}

func (b *Broker) GetCatalog(c *broker.RequestContext) (*osb.CatalogResponse, error) {
	services, err := b.Client.ListServices()
	if err != nil {
		return nil, err
	}

	response := &osb.CatalogResponse{
		Services: services,
	}

	return response, nil
}

func (b *Broker) Provision(request *osb.ProvisionRequest, c *broker.RequestContext) (*osb.ProvisionResponse, error) {
	b.Lock()
	defer b.Unlock()

	namespace := fmt.Sprintf("%v", request.Context["namespace"])
	err := b.Client.Provision(request.InstanceID, request.ServiceID, request.PlanID, namespace)
	if err != nil {
		glog.Errorln(err)
		return nil, err
	}

	response := osb.ProvisionResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *Broker) Deprovision(request *osb.DeprovisionRequest, c *broker.RequestContext) (*osb.DeprovisionResponse, error) {
	b.Lock()
	defer b.Unlock()

	err := b.Client.Deprovision(request.InstanceID)
	if err != nil {
		glog.Errorln(err)
		return nil, err
	}

	response := osb.DeprovisionResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *Broker) LastOperation(request *osb.LastOperationRequest, c *broker.RequestContext) (*osb.LastOperationResponse, error) {
	// Your last-operation business logic goes here

	return nil, nil
}

func (b *Broker) Bind(request *osb.BindRequest, c *broker.RequestContext) (*osb.BindResponse, error) {
	b.Lock()
	defer b.Unlock()

	creds, err := b.Client.Bind(request.InstanceID)
	if err != nil {
		glog.Errorln(err)
		return nil, err
	}

	response := osb.BindResponse{
		Credentials: creds,
	}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *Broker) Unbind(request *osb.UnbindRequest, c *broker.RequestContext) (*osb.UnbindResponse, error) {
	// nothing to do

	response := osb.UnbindResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *Broker) Update(request *osb.UpdateInstanceRequest, c *broker.RequestContext) (*osb.UpdateInstanceResponse, error) {
	// Not supported, do nothing

	response := osb.UpdateInstanceResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *Broker) ValidateBrokerAPIVersion(version string) error {
	return nil
}
