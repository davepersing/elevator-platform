package scheduler

import (
	"fmt"
	"math"

	"github.com/davepersing/elevator-platform/elevator"
	"github.com/davepersing/elevator-platform/passenger"
	"github.com/davepersing/elevator-platform/util"
)

// Based on the statuses returned from each elevator, makes a decision on where to schedule the elevator.
// Returns the elevatorId and groupId of the assigned elevator.
func FindElevator(statuses map[int]*elevator.ElevatorStatus, p *passenger.Passenger) (int, int) {

	// Take out all statuses that aren't available.
	statusResults := filterUnavailableStatuses(statuses)

	// If all are unavailable, bail out early.
	if len(statusResults) == 0 {
		return -1, -1
	}

	closestIdleId := getClosestIdle(statusResults, p)
	closestDirectionalId := getClosestDirectionalId(statusResults, p)

	// Found the two closest, if both IDs are >= 0,
	// then we have a solution for both and compare the two to find which is closest.
	//
	// If one is >= 0 and the other -1, take one.
	//
	// If both are -1, assign to random?  Come up with a better way than random.
	// Elevators at extremes are more likely to hit the top and head back down?
	//
	if closestIdleId >= 0 && closestDirectionalId >= 0 {

		// We have a solution for both, so compare and find out which is closer.
		idleStatus := statusResults[closestIdleId]
		directionalStatus := statusResults[closestDirectionalId]

		idleTest := util.Abs(idleStatus.CurrentFloor - p.CurrentFloor)
		directionalTest := util.Abs(directionalStatus.CurrentFloor - p.CurrentFloor)

		if idleTest < directionalTest {

			return closestIdleId, groupIdForId(statusResults, closestIdleId)
		} else {

			return closestDirectionalId, groupIdForId(statusResults, closestDirectionalId)
		}
	} else if closestIdleId >= 0 {

		return closestIdleId, groupIdForId(statusResults, closestIdleId)
	} else if closestDirectionalId >= 0 {

		return closestDirectionalId, groupIdForId(statusResults, closestDirectionalId)
	}

	// If we got this far, there are no idle and no moving the same direction.
	// So calculate the closest based on the elevator's current target floor.
	// Which ever is closest to the passenger wins.
	closestIdToPassenger := getClosestToTargetId(statusResults, p)

	// All elevators are in maintenance or error states.
	if closestIdToPassenger < 0 {
		return -1, -1
	}

	return closestIdToPassenger, statusResults[closestIdToPassenger].GroupId
}

// Convenience method to grab the group ID out of the statuses hash.
func groupIdForId(statuses map[int]*elevator.ElevatorStatus, id int) int {
	return statuses[id].GroupId
}

// Gets the closest elevator to the passenger by checking the elevator's current target floor.
// Target floor is defined as the highest or lowest point it needs to go based on the passenger's
// destination floor.
func getClosestToTargetId(statuses map[int]*elevator.ElevatorStatus, p *passenger.Passenger) int {

	var closestIdToPassenger int
	closestTarget := math.MaxInt32

	for _, status := range statuses {

		test := util.Abs(status.CurrentTargetFloor - p.CurrentFloor)
		// less than or equal because the elevators could be heading to the same floor.
		if test < closestTarget {
			closestIdToPassenger = status.Id
			closestTarget = test
		}
	}
	return closestIdToPassenger
}

// Gets the closest elevator based on an elevator direction
// If the elevator is moving in the same direction the passenger wishes to go and  the `abs(elevator.CurrentFloor - passenger.CurrentFloor)`
// Whichever elevator is moving in the same direction and has the fewest number of floors between it and the passenger will be chosen.
func getClosestDirectionalId(statuses map[int]*elevator.ElevatorStatus, p *passenger.Passenger) int {

	var direction int = -1
	if p.CurrentFloor > p.DestinationFloor {
		direction = elevator.STATE_MOVING_DOWN
	} else {
		direction = elevator.STATE_MOVING_UP
	}

	sameDirection := make(map[int]*elevator.ElevatorStatus)
	for id, es := range statuses {
		if es.CurrentState == direction {
			sameDirection[id] = es
		}
	}

	if len(sameDirection) == 0 {
		return -1
	}

	// Only a single elevator moving in the same direction, so return it.
	// Could still be moving in the same direction, but BELOW the current target.
	if len(sameDirection) == 1 {
		for id, es := range sameDirection {

			if (direction == elevator.STATE_MOVING_UP && p.CurrentFloor > es.CurrentFloor) ||
				(direction == elevator.STATE_MOVING_DOWN && p.CurrentFloor < es.CurrentFloor) {
				return id
			} else {
				return -1
			}
		}
	}

	var closestId int
	var closestInFloors int = math.MaxInt32
	for id, es := range sameDirection {
		test := util.Abs(es.CurrentFloor - p.CurrentFloor)
		if test < closestInFloors {
			closestId = id
			closestInFloors = test
			fmt.Printf("DIRECTIONAL: Setting closestID: %d, floors: %d\n", id, closestInFloors)
		}
	}

	return closestId
}

// Returns the closest idling elevator.
// If the number of idlers is greater than 1, then calculate which idling elevator is closest to the passenger.
// based on the same formula of abs(elevator.CurrentFloor - p.CurrentFloor).  Whichever value is lowest, wins.
func getClosestIdle(statuses map[int]*elevator.ElevatorStatus, p *passenger.Passenger) int {

	idlers := make(map[int]*elevator.ElevatorStatus)
	for id, es := range statuses {
		if es.CurrentState == elevator.STATE_IDLE {
			idlers[id] = es
		}
	}

	// No idling elevators, so bail out.
	if len(idlers) == 0 {
		return -1
	}

	// Only one elevator idling, so return it.
	if len(idlers) == 1 {
		for id, _ := range idlers {
			return id
		}
	}

	var closestInFloors int = math.MaxInt32
	var closestId int
	for id, es := range idlers {
		test := util.Abs(es.CurrentFloor - p.CurrentFloor)
		if test < closestInFloors {
			closestId = id
			closestInFloors = test
		}
	}

	return closestId
}

// Filters out any elevator with a non-available status.
// Returns a map of elevator statuses.
func filterUnavailableStatuses(statuses map[int]*elevator.ElevatorStatus) map[int]*elevator.ElevatorStatus {
	for id, es := range statuses {
		var remove bool

		// Check the elevator state.
		switch es.CurrentState {
		case elevator.STATE_ERROR,
			elevator.STATE_MAINTENANCE:
			remove = true
		}

		// Check the capacity.
		// TODO: Fix this.  MaxCapacity needs to be in here somewhere.
		// Also, if all elevators are at max capacity, still need to schedule passengers.
		// if len(es.Passengers) >= MAX_CAPACITY {
		// 	remove = true
		// }

		if remove {
			delete(statuses, id)
		}
	}
	return statuses
}
