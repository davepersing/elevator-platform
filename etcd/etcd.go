package etcd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type (
	// Contains members needed to connect to Etcd cluster
	//and references to an instance of the keys API with a client.
	Etcd struct {
		Url     string // Url to the Etcd cluster.
		KeysApi client.KeysAPI
		Client  client.Client
	}
)

// Initializes the Etcd module.
func (e *Etcd) Init() {
	config := client.Config{
		Endpoints:               []string{e.Url},
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	c, err := client.New(config)
	if err != nil {
		fmt.Printf("Cannot connect to etcd. Error: %v\n", err)
		return
	}

	e.Client = c
	e.KeysApi = client.NewKeysAPI(c)
}

// Returns all statuses from the /elevator_status endpoint
func (e *Etcd) GetAllStatuses() ([]*client.Node, error) {
	resp, err := e.KeysApi.Get(context.Background(), "/elevator_status", nil)
	if err != nil {
		fmt.Printf("Cannot get all statuses.  Error: %+v\n", err)
		return nil, err
	}

	return resp.Node.Nodes, nil
}

// Sets a passenger to the waiting key in etcd.  When this is set, the listening elevator will be notified.
func (e *Etcd) SetPassenger(elevatorId, groupId int, jsonData []byte) error {
	path := "/wait/" + strconv.Itoa(groupId) + "-" + strconv.Itoa(elevatorId)

	// TODO:  This needs to be fixed to include additionally scheduled passengers if the elevator hasn't
	// been able to process the one that was there before.

	_, err := e.KeysApi.Set(context.Background(), path, string(jsonData), nil)
	if err != nil {
		fmt.Printf("Error setting passenger to etcd.  Error: %s\n", err.Error())
		return err
	}
	return nil
}

func (e *Etcd) SetMaintenanceMode(elevatorId, groupId, maintenance string) error {
	path := "/maintenance/" + groupId + "-" + elevatorId

	if _, err := e.KeysApi.Set(context.Background(), path, maintenance, nil); err != nil {
		fmt.Printf("Error setting maintenance mode in etcd.  Error :%s\n", err.Error())
		return err
	}
	return nil
}
