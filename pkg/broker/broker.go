package broker

import (
	"errors"
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
	mb := minibroker.NewClient(o.HelmRepoUrl, o.ServiceCatalogEnabledOnly)
	err := mb.Init()
	if err != nil {
		return nil, err
	}

	// For example, if your Broker requires a parameter from the command
	// line, you would unpack it from the Options and set it on the
	// Broker here.
	return &Broker{
		Client:           mb,
		async:            true,
		defaultNamespace: o.DefaultNamespace,
	}, nil
}

// Broker provides an implementation of broker.Interface
type Broker struct {
	Client *minibroker.Client

	// Indiciates if the broker should handle the requests asynchronously.
	async bool
	// Synchronize go routines.
	sync.RWMutex
	// Default namespace to run brokers if not specified during request
	defaultNamespace string
}

var _ broker.Interface = &Broker{}

func (b *Broker) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	services, err := b.Client.ListServices()
	if err != nil {
		return nil, err
	}

	response := &broker.CatalogResponse{
		CatalogResponse: osb.CatalogResponse{
			Services: services,
		},
	}

	return response, nil
}

func (b *Broker) Provision(request *osb.ProvisionRequest, c *broker.RequestContext) (*broker.ProvisionResponse, error) {
	b.Lock()
	defer b.Unlock()

	namespace := b.defaultNamespace
	if request.Context["namespace"] != nil {
		namespace = request.Context["namespace"].(string)
	}

	if namespace == "" {
		err := errors.New("Cannot provision with empty namespace")
		glog.Errorln(err)
		return nil, err
	}

	glog.V(5).Infof("Provisioning %s (%s/%s) in %s", request.InstanceID, request.ServiceID, request.PlanID, namespace)

	operationName, err := b.Client.Provision(request.InstanceID, request.ServiceID, request.PlanID, namespace, request.AcceptsIncomplete, request.Parameters)
	if err != nil {
		glog.Errorln(err)
		return nil, err
	}

	response := broker.ProvisionResponse{}
	if request.AcceptsIncomplete {
		response.Async = true
		operationKey := osb.OperationKey(operationName)
		response.OperationKey = &operationKey
	}

	glog.V(5).Infof("Successfully initiated provisioning %s (%s/%s) in %s", request.InstanceID, request.ServiceID, request.PlanID, namespace)
	return &response, nil
}

func (b *Broker) Deprovision(request *osb.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	glog.V(5).Infof("Deprovisioning %s (%s/%s)", request.InstanceID, request.ServiceID, request.PlanID)
	b.Lock()
	defer b.Unlock()

	operationName, err := b.Client.Deprovision(request.InstanceID, request.AcceptsIncomplete)
	if err != nil {
		glog.Errorln(err)
		return nil, err
	}

	response := broker.DeprovisionResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
		operationKey := osb.OperationKey(operationName)
		response.OperationKey = &operationKey
	}

	glog.V(5).Infof("Successfully initiated deprovisioning %s (%s/%s)", request.InstanceID, request.ServiceID, request.PlanID)
	return &response, nil
}

// LastOperation provides information on the state of the last asynchronous operation
func (b *Broker) LastOperation(request *osb.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	glog.V(5).Infof("Getting last operation of %s (%v/%v)", request.InstanceID, request.ServiceID, request.PlanID)
	b.Lock()
	defer b.Unlock()

	response, err := b.Client.LastOperationState(request.InstanceID, request.OperationKey)
	if err != nil {
		glog.Errorln(err)
		return nil, err
	}

	wrappedResponse := broker.LastOperationResponse{LastOperationResponse: *response}

	glog.V(5).Infof("Successfully got last operation of %s (%v/%v): %+v", request.InstanceID, request.ServiceID, request.PlanID, response)
	return &wrappedResponse, nil
}

func (b *Broker) Bind(request *osb.BindRequest, c *broker.RequestContext) (*broker.BindResponse, error) {
	glog.V(5).Infof("Binding %s (%s)", request.InstanceID, request.ServiceID)
	b.Lock()
	defer b.Unlock()

	creds, err := b.Client.Bind(request.InstanceID, request.ServiceID, request.Parameters)
	if err != nil {
		glog.Errorln(err)
		return nil, err
	}

	response := broker.BindResponse{
		BindResponse: osb.BindResponse{
			Credentials: creds,
		},
	}
	if request.AcceptsIncomplete {
		response.Async = false // We do not currently accept asynchronous operations on bind
	}

	glog.V(5).Infof("Successfully binding %s (%s)", request.InstanceID, request.ServiceID)

	return &response, nil
}

func (b *Broker) Unbind(request *osb.UnbindRequest, c *broker.RequestContext) (*broker.UnbindResponse, error) {
	glog.V(5).Infof("Unbinding %s (%s)", request.InstanceID, request.ServiceID)
	// nothing to do

	response := broker.UnbindResponse{}
	if request.AcceptsIncomplete {
		response.Async = false // We do not currently accept asynchronous operations on unbind
	}

	glog.V(5).Infof("Successfully unbinding %s (%s)", request.InstanceID, request.ServiceID)
	return &response, nil
}

func (b *Broker) Update(request *osb.UpdateInstanceRequest, c *broker.RequestContext) (*broker.UpdateInstanceResponse, error) {
	// Not supported, do nothing

	response := broker.UpdateInstanceResponse{}
	if request.AcceptsIncomplete {
		response.Async = false // We do not currently accept asynchronous operations on update
	}

	return &response, nil
}

func (b *Broker) ValidateBrokerAPIVersion(version string) error {
	return nil
}
