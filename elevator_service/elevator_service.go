package elevator_service

import (
	"github.com/davepersing/elevator-platform/elevator"
	"github.com/davepersing/elevator-platform/http_api"
)

type (
	// Elevator Service unifies all the disparate parts of the elevator microservice.
	// This contains an HTTP API that listens for incoming requests and can send them to the elevator for scheduling.
	// An instance of an elevator that runs independently of any other elevator, but will respond to passenger requests.
	ElevatorService struct {
		Elevator *elevator.Elevator
		HttpApi  *http_api.HttpApi
	}
)

// INitializes the overall elevator service.
func (es *ElevatorService) Init() {
	// Initialize the elevator API.
	es.HttpApi.Init()
	es.Elevator.Init()
}
