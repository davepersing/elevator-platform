package http_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/coreos/etcd/client"
	"github.com/davepersing/elevator-platform/elevator"
	"github.com/davepersing/elevator-platform/etcd"
	"github.com/davepersing/elevator-platform/passenger"
	"github.com/davepersing/elevator-platform/scheduler"
	"golang.org/x/net/context"
)

type (
	HttpApi struct {
		Hostname string // Hostname this server listens on.
		Port     string // Port this http server listens on.
		*etcd.Etcd
	}
)

// Initializes the HTTP API module.
func (ha *HttpApi) Init() {
	ha.Etcd.Init()

	if ha.Hostname == "" {
		fmt.Println("Hostname is empty.  Running on localhost.")
	}

	if ha.Port == "" {
		panic("HttpApi Port must be specified.")
	}

	go func(ha *HttpApi) {

		mux := http.NewServeMux()
		mux.HandleFunc("/elevator_call", ha.handleElevatorCall)
		http.ListenAndServe(ha.Hostname+ha.Port, mux)
	}(ha)
}

// Handles the passenger's request for an elevator.
func (ha *HttpApi) handleElevatorCall(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var p passenger.Passenger

	if err := decoder.Decode(&p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error decoding passenger struct: %v", err)
		return
	}

	statuses, err := ha.Etcd.GetAllStatuses()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("Error getting all statuses.  Error: %v", err)
		return
	}

	elevatorStatuses, err := decodeStatuses(statuses)
	if err != nil {
		fmt.Printf("Error decoding statuses from etcd.  Error: %v\n", err)
	}

	elevatorId, groupId := scheduler.FindElevator(elevatorStatuses, &p)

	jsonBytes, err := json.Marshal(&p)
	if err != nil {
		fmt.Printf("Could not marshal passenger json.  Error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = ha.Etcd.SetPassenger(elevatorId, groupId, jsonBytes)
	if err != nil {
		fmt.Printf("Could not set passenger to %d\n", elevatorId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Update etcd with the status letting the elevator know it's status has change.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	success := map[string]int{"elevatorId": elevatorId}

	if err := json.NewEncoder(w).Encode(success); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error sending successful response.  %v\n", err)
	}
}

func decodeStatuses(statuses []*client.Node) (map[int]*elevator.ElevatorStatus, error) {

	elStatuses := make(map[int]*elevator.ElevatorStatus)

	for _, node := range statuses {
		var elStat elevator.ElevatorStatus
		err := json.Unmarshal([]byte(node.Value), &elStat)
		if err != nil {
			// skip this.
			// Don't add to hash if the response can't be deciphered.
		} else {
			elStatuses[elStat.Id] = &elStat
		}
	}

	return elStatuses, nil
}

func (ha *HttpApi) getAllStatuses() ([]*client.Node, error) {
	resp, err := ha.KeysApi.Get(context.Background(), "/elevators", nil)
	if err != nil {
		fmt.Printf("Cannot get all statuses.  Error: %+v\n", err)
		return nil, err
	}

	return resp.Node.Nodes, nil
}

func (ha *HttpApi) setPassenger(elevatorId, groupId int, jsonData []byte) error {
	path := "/wait/" + strconv.Itoa(groupId) + "-" + strconv.Itoa(elevatorId)
	_, err := ha.KeysApi.Set(context.Background(), path, string(jsonData), nil)
	if err != nil {
		fmt.Printf("Error setting passenger to etcd.  Error: %s\n", err.Error())
		return err
	}
	return nil
}
