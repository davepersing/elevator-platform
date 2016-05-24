package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

type SuccessResult struct {
	ElevatorId int `json:"elevatorId"`
}

// Sends a POST request to a given URL with desired current and destination floors.
// Creates a new passenger, serializes into JSON, and POSTs to the given endpoint.
func SendPassengerPost(url string, currFloor, destFloor int) (int, error) {
	// Send the request to the random known node pool.
	p := passenger.Passenger{CurrentFloor: currFloor, DestinationFloor: destFloor}
	data, err := json.Marshal(p)
	req, err := http.NewRequest("POST", "http://localhost"+url+"/elevator_call", bytes.NewBuffer(data))
	if err != nil {
		return -1, err
	}

	// Set the JSON header.
	req.Header.Set("Content-Type", "application/json")

	// Make the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error from url %s.  Error: %s\n", url, err.Error())
		return -1, err
	}
	defer resp.Body.Close()

	// Get the elevator to take.
	decoder := json.NewDecoder(resp.Body)
	var result SuccessResult
	err = decoder.Decode(&result)
	if err != nil {
		fmt.Printf("Could not decode result of elevator request.  Error: %v\n", err)
		return -1, err
	}

	return result.ElevatorId, nil
}
