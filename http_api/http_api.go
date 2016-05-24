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

	maintenanceRequest struct {
		ElevatorId  string `json:"elevatorId"`
		GroupId     string `json:"groupId"`
		Maintenance string `json:"maintenance"`
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
		mux.HandleFunc("/maintenance", ha.handleElevatorMaintenance)
		http.ListenAndServe(ha.Hostname+ha.Port, mux)
	}(ha)
}

// Handles request to put elevator in maintenance mode.
//
func (ha *HttpApi) handleElevatorMaintenance(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var mr maintenanceRequest

	if err := decoder.Decode(&mr); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("Error decoding maintenance struct: %v\n", err)
		fmt.Fprintf(w, "Error decoding maintenance struct: %v\n", err)
		return
	}

	if err := ha.Etcd.SetMaintenanceMode(mr.ElevatorId, mr.GroupId, mr.Maintenance); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error setting maintenance mode for elevator: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	success := map[string]string{
		"elevatorId":  mr.ElevatorId,
		"groupId":     mr.GroupId,
		"maintenance": mr.Maintenance,
	}

	if err := json.NewEncoder(w).Encode(success); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error sending setMaintenanceResponse: %v\n", err)
	}
}

// Handles the passenger's request for an elevator.
func (ha *HttpApi) handleElevatorCall(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var p passenger.Passenger

	if err := decoder.Decode(&p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error decoding passenger struct: %v\n", err)
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

	if elevatorId < 0 || groupId < 0 {
		fmt.Println("Could not schedule passenger.  All elevators are busy.")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

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

	success := map[string]string{"elevatorId": strconv.Itoa(elevatorId), "groupId": strconv.Itoa(groupId)}

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
