package elevator_test

import (
	"testing"

	"github.com/davepersing/elevator-platform/elevator"
	"github.com/davepersing/elevator-platform/passenger"
)

func TestNewElevator(t *testing.T) {
	e := elevator.Elevator{
		MaxFloor:    16,
		MinFloor:    1,
		MaxCapacity: 16,
		ElevatorStatus: elevator.ElevatorStatus{
			Id:           0,
			GroupId:      0,
			CurrentFloor: 1,
			CurrentState: elevator.STATE_IDLE,

			WaitingPassengers: elevator.WaitingPassengers{Waiting: make([]*passenger.Passenger, 0)},
			Passengers:        make([]*passenger.Passenger, 0),
		},
	}

	if e.MaxFloor != 16 || e.MinFloor != 1 {
		t.Errorf("Did not create elevator with correct parameters. %v\n", e)
	}
}
