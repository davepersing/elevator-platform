package scheduler

import (
	_ "fmt"
	"testing"

	"github.com/davepersing/elevator-platform/elevator"
	"github.com/davepersing/elevator-platform/passenger"
)

func TestGetClosestIdle1(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 1,
		CurrentState: elevator.STATE_IDLE,
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 16,
		CurrentState: elevator.STATE_IDLE,
	}

	closestId := getClosestIdle(statuses, &passenger.Passenger{CurrentFloor: 8, DestinationFloor: 16})
	// Should be elevator 0.
	if closestId != 0 {
		t.Fail()
	}
}

func TestClosestIdle2(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 1,
		CurrentState: elevator.STATE_IDLE,
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 16,
		CurrentState: elevator.STATE_IDLE,
	}

	closestId := getClosestIdle(statuses, &passenger.Passenger{CurrentFloor: 16, DestinationFloor: 1})
	if closestId != 1 {
		t.Fail()
	}
}

func TestClosestIdle3(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 4,
		CurrentState: elevator.STATE_MOVING_UP,
		Passengers:   []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 1, DestinationFloor: 16}},
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 16,
		CurrentState: elevator.STATE_IDLE,
	}

	closestId := getClosestIdle(statuses, &passenger.Passenger{CurrentFloor: 16, DestinationFloor: 1})
	if closestId != 1 {
		t.Errorf("Got %d but wanted 1", closestId)
	}
}

func TestClosestIdle4(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 4,
		CurrentState: elevator.STATE_MOVING_UP,
		Passengers:   []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 1, DestinationFloor: 16}},
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 16,
		CurrentState: elevator.STATE_LOADING,
	}

	closestId := getClosestIdle(statuses, &passenger.Passenger{CurrentFloor: 16, DestinationFloor: 1})
	if closestId != -1 {
		t.Errorf("Got %d but wanted 1", closestId)
	}
}

func BenchmarkClosestIdle(b *testing.B) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 4,
		CurrentState: elevator.STATE_MOVING_UP,
		Passengers:   []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 1, DestinationFloor: 16}},
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 16,
		CurrentState: elevator.STATE_IDLE,
	}

	for i := 0; i < b.N; i++ {
		getClosestIdle(statuses, &passenger.Passenger{CurrentFloor: 16, DestinationFloor: 1})
	}
}

// ====================== Closest Directional ========================

func TestClosestDirectionalId1(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 4,
		CurrentState: elevator.STATE_MOVING_UP,
		Passengers:   []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 1, DestinationFloor: 16}},
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 16,
		CurrentState: elevator.STATE_IDLE,
	}

	closestId := getClosestDirectionalId(statuses, &passenger.Passenger{CurrentFloor: 5, DestinationFloor: 16})
	if closestId != 0 {
		t.Errorf("Got %d but wanted 0", closestId)
	}
}

func TestClosestDirectionalId2(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 4,
		CurrentState: elevator.STATE_MOVING_DOWN,
		Passengers:   []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 1, DestinationFloor: 16}},
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 4,
		CurrentState: elevator.STATE_MOVING_UP,
	}

	closestId := getClosestDirectionalId(statuses, &passenger.Passenger{CurrentFloor: 5, DestinationFloor: 16})
	if closestId != 1 {
		t.Errorf("Got %d but wanted 1", closestId)
	}
}

func TestClosestDirectionalId3(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:           0,
		GroupId:      0,
		CurrentFloor: 6,
		CurrentState: elevator.STATE_MOVING_DOWN,
		Passengers:   []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 1, DestinationFloor: 16}},
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:           1,
		GroupId:      0,
		CurrentFloor: 4,
		CurrentState: elevator.STATE_MOVING_UP,
	}

	closestId := getClosestDirectionalId(statuses, &passenger.Passenger{CurrentFloor: 5, DestinationFloor: 1})
	if closestId != 0 {
		t.Errorf("Got %d but wanted 0", closestId)
	}
}

// ====================== Closest Directional ========================

// Tests the permutation where both elevators are moving in the same direction and there are no IDLE.
/*
If both elevators are moving down
  E1	E2
10| |	| | d
09| |	| | ^
08|e| | | ^
07| | | | ^
06| | | | ^
05| |	| | ^
04| | |e| ^
03| | | | p
02| | | |
01| | | |

Elevator E2 is moving to floor 1.
Elevator E1 is moving to floor 4.
Passenger P wants to get on at floor 3.

Since E1 will be the closest at it's target of 4, choose E1.
e1.CurrentTargetFloor - p.CurrentFloor = abs(4 - 3) = 1
e2.CurrentTargetFloor - p.CurrentFloor = abs(1 - 3) = 2

1 < 2, so take elevator E1.
*/
func TestClosestTargetFloor1(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:                 0,
		GroupId:            0,
		CurrentFloor:       8,
		CurrentState:       elevator.STATE_MOVING_DOWN,
		CurrentTargetFloor: 4,
		Passengers:         []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 10, DestinationFloor: 4}},
	}
	statuses[1] = &elevator.ElevatorStatus{
		Id:                 1,
		GroupId:            0,
		CurrentFloor:       4,
		CurrentState:       elevator.STATE_MOVING_DOWN,
		CurrentTargetFloor: 1,
		Passengers:         []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 8, DestinationFloor: 1}},
	}

	closestId := getClosestToTargetId(statuses, &passenger.Passenger{CurrentFloor: 3, DestinationFloor: 10})
	if closestId != 0 {
		t.Errorf("Got %d but wanted 0", closestId)
	}
}

// Test the opposite direction.
func TestClosestTargetFloor2(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:                 0,
		GroupId:            0,
		CurrentFloor:       8,
		CurrentState:       elevator.STATE_MOVING_UP,
		CurrentTargetFloor: 16,
		Passengers:         []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 10, DestinationFloor: 16}},
	}

	statuses[1] = &elevator.ElevatorStatus{
		Id:                 1,
		GroupId:            0,
		CurrentFloor:       4,
		CurrentState:       elevator.STATE_MOVING_UP,
		CurrentTargetFloor: 12,
		Passengers:         []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 8, DestinationFloor: 12}},
	}

	closestId := getClosestToTargetId(statuses, &passenger.Passenger{CurrentFloor: 3, DestinationFloor: 10})
	if closestId != 1 {
		t.Errorf("Got %d but wanted 1", closestId)
	}
}

// Make sure this still works if the elevators end up on the same floor.
// Since these are heading to the same destination, make sure it's either id since it doesn't matter.
func TestClosestTargetFloor3(t *testing.T) {
	statuses := make(map[int]*elevator.ElevatorStatus)
	statuses[0] = &elevator.ElevatorStatus{
		Id:                 0,
		GroupId:            0,
		CurrentFloor:       8,
		CurrentState:       elevator.STATE_MOVING_UP,
		CurrentTargetFloor: 16,
		Passengers:         []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 10, DestinationFloor: 16}},
	}

	statuses[1] = &elevator.ElevatorStatus{
		Id:                 1,
		GroupId:            0,
		CurrentFloor:       4,
		CurrentState:       elevator.STATE_MOVING_UP,
		CurrentTargetFloor: 16,
		Passengers:         []*passenger.Passenger{&passenger.Passenger{CurrentFloor: 8, DestinationFloor: 16}},
	}

	closestId := getClosestToTargetId(statuses, &passenger.Passenger{CurrentFloor: 3, DestinationFloor: 10})
	if closestId < 0 {
		t.Errorf("Got %d but wanted >= 0", closestId)
	}
}
