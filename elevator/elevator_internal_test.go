package elevator

import (
	"testing"

	"github.com/davepersing/elevator-platform/passenger"
)

func TestMoveUpWithWaitingPassenger(t *testing.T) {
	e := getBaseElevator()

	ok := e.addNewWaitingPassenger(&passenger.Passenger{CurrentFloor: 2, DestinationFloor: 4})
	if !ok {
		t.Error("Error adding passenger to waiting list.")
	}

	e.move()

	if e.CurrentState != STATE_MOVING_UP {
		t.Error("Elevator should be moving up.  Passenger is waiting on floor 2")
	}
}

func TestMoveDownWithWaitingPassenger(t *testing.T) {
	e := getBaseElevator()
	e.CurrentFloor = 5

	ok := e.addNewWaitingPassenger(&passenger.Passenger{CurrentFloor: 2, DestinationFloor: 1})
	if !ok {
		t.Error("Error adding passenger to waiting list.")
	}

	e.move()

	if e.CurrentState != STATE_MOVING_DOWN {
		t.Errorf("Expected state is STATE_MOVING_DOWN but got %d", e.CurrentState)
	}
}

func TestLoadPassengerAndMoveDown(t *testing.T) {
	e := getBaseElevator()
	e.CurrentFloor = 3

	ok := e.addNewWaitingPassenger(&passenger.Passenger{CurrentFloor: 3, DestinationFloor: 1})
	if !ok {
		t.Error("Elevator should have loaded passenger.")
	}

	e.move()

	if e.CurrentState != STATE_LOADING {
		t.Errorf("Expected state is STATE_LOADING, but got %d", e.CurrentState)
	}

	e.move()
	if len(e.Passengers) <= 0 {
		t.Error("Passenger should have been loaded.")
	}

	if e.CurrentState != STATE_MOVING_DOWN {
		t.Errorf("Expected state to be STATE_MOVING_DOWN, but got %d", e.CurrentState)
	}
}

func TestUnloadPassengerAndMoveUpToWaitingPassenger(t *testing.T) {
	e := getBaseElevator()
	e.CurrentState = STATE_MOVING_DOWN
	e.CurrentFloor = 3
	e.addNewPassenger(&passenger.Passenger{CurrentFloor: 8, DestinationFloor: 2})

	ok := e.addNewWaitingPassenger(&passenger.Passenger{CurrentFloor: 3, DestinationFloor: 1})
	if !ok {
		t.Error("Should have added passenger to waiting list.")
	}

	e.move()

	if e.CurrentState != STATE_UNLOADING {
		t.Errorf("State should be STATE_UNLOADING but got %d\n", e.CurrentState)
	}

	e.move()
	if len(e.Passengers) > 0 {
		t.Error("Should have unloaded all passengers.")
	}

	if e.CurrentState != STATE_MOVING_UP {
		t.Errorf("State should be STATE_MOVING_UP but got %d\n", e.CurrentState)
	}
}

func getBaseElevator() *Elevator {
	return &Elevator{
		MaxFloor:    16,
		MinFloor:    1,
		MaxCapacity: 16,
		ElevatorStatus: ElevatorStatus{
			Id:           0,
			GroupId:      0,
			CurrentFloor: 1,
			CurrentState: STATE_IDLE,

			WaitingPassengers: WaitingPassengers{Waiting: make([]*passenger.Passenger, 0)},
			Passengers:        make([]*passenger.Passenger, 0),
		},
	}
}
