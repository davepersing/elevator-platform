package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/davepersing/elevator-platform/passenger"
)

// Returns a "random" index based on modding the current unix Epoch with the total number of elevators.
func GetRandomIndex(max int) int {
	if max > 1 {
		// Definitely not random, but a reasonable distribution across nodes.
		return int(time.Now().Unix() % int64(max))
	}
	return 0
}

// Returns the absolute value of an integer.
func Abs(n int) int {
	if n < 0 {
		n = -n
	}
	return n
}

type (
	SuccessResult struct {
		ElevatorId string `json:"elevatorId"`
		GroupId    string `json:"groupId"`
	}

	Maintenance struct {
		ElevatorId  string `json:"elevatorId"`
		GroupId     string `json:"groupId"`
		Maintenance string `json:"maintenance"`
	}
)

func SendMaintenancePost(port string, elevatorId, groupId int, maintenance bool) (string, string, error) {
	mr := Maintenance{
		ElevatorId:  strconv.Itoa(elevatorId),
		GroupId:     strconv.Itoa(groupId),
		Maintenance: strconv.FormatBool(maintenance)}

	data, err := json.Marshal(mr)
	if err != nil {
		fmt.Printf("Could not marshal maintenance request: %v\n", err)
		return "", "", err
	}

	req, err := http.NewRequest("POST", "http://localhost"+port+"/maintenance", bytes.NewBuffer(data))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending maintenance request: %s\v", err.Error())
		return "", "", err
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var result Maintenance
	err = decoder.Decode(&result)
	if err != nil {
		fmt.Printf("Could not decode result of maintenance request: %v\n", err)
		return "", "", err
	}

	return result.ElevatorId, result.GroupId, nil
}

// Sends a POST request to a given URL with desired current and destination floors.
// Creates a new passenger, serializes into JSON, and POSTs to the given endpoint.
func SendPassengerPost(port string, currFloor, destFloor int) (string, string, error) {
	// Send the request to the random known node pool.
	p := passenger.Passenger{CurrentFloor: currFloor, DestinationFloor: destFloor}
	data, err := json.Marshal(p)
	req, err := http.NewRequest("POST", "http://localhost"+port+"/elevator_call", bytes.NewBuffer(data))
	if err != nil {
		return "", "", err
	}

	// Set the JSON header.
	req.Header.Set("Content-Type", "application/json")

	// Make the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error from port %s.  Error: %s\n", port, err.Error())
		return "", "", err
	}
	defer resp.Body.Close()

	// Get the elevator to take.
	decoder := json.NewDecoder(resp.Body)
	var result SuccessResult
	err = decoder.Decode(&result)
	if err != nil {
		fmt.Printf("Could not decode result of elevator request.  Error: %v\n", err)
		return "", "", err
	}

	return result.ElevatorId, result.GroupId, nil
}
