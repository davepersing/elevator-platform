package elevator

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/davepersing/elevator-platform/etcd"
	"github.com/davepersing/elevator-platform/passenger"
	"golang.org/x/net/context"
)

const (
	// Elevator states
	STATE_IDLE = iota
	STATE_MOVING_UP
	STATE_MOVING_DOWN
	STATE_MAINTENANCE
	STATE_LOADING
	STATE_UNLOADING
	STATE_ERROR // This is really just convenience to set the response if the request times out to this node.
)

type (
	// Defines an elevator.  Contains a current elevator status,
	// information about the server, and known nodes in the cluster.
	Elevator struct {
		MaxFloor    int // Elevator could have a max floor that is less than actual floor max.
		MinFloor    int // Elevators could be split into different banks on different floors.
		MaxCapacity int // Max number of persons currently, not a hard weight limit.

		ElevatorStatus // Current state of the elevator

		*etcd.Etcd // Etcd
	}

	// Defines a status for a given elevator.
	ElevatorStatus struct {
		DisplayId int `json:"displayId"` // 1 - n instead of 0 - (n - 1)

		Id      int `json:"id"` // Int ID of the elevator.
		GroupId int `json:"groupId"`

		CurrentFloor       int `json:"currentFloor"`       // The floor the elevator is currently on.
		CurrentState       int `json:"currentState"`       // Tracks the current state of the elevator.
		CurrentTargetFloor int `json:"currentTargetFloor"` // Tracks the current highest/lowest floor the elevator is going to.

		WaitingPassengers

		Passengers []*passenger.Passenger `json:"passengers"` // The current number of passengers actually on the elevator.
	}

	WaitingPassengers struct {
		sync.Mutex `json:"-"`
		Waiting    []*passenger.Passenger `json:"waitingPassengers"` // The current number of waiting passengers on other floors.
	}
)

// Initializes the elevator with the passed-in parameters.
// Starts two goroutines.
// - One to run the timer to make an elevator move.
// - One to start the scheduling server
func (e *Elevator) Init() {
	e.Etcd.Init()

	// Grab the possibly existing data from etcd.
	e.loadExistingStatus()

	e.startPassengerWatcher()

	e.startMaintenanceWatcher()

	e.startTimerLoop()
}

// Starts a timer to report status every one second and move the elevator to the next step.
func (e *Elevator) startTimerLoop() {

	go func(e *Elevator) {
		c := time.Tick(1 * time.Second)
		for range c {
			// To get the ordered and pretty output, save the current state and add the moved state.
			currStatus := "Current: " + e.prettyPrintStatus()
			e.move()
			e.saveState()
			currStatus += "After:   " + e.prettyPrintStatus()
			fmt.Println(currStatus)
		}
	}(e)
}

// Start a watcher to deal with adding new passengers.
// TODO:  This goroutine crashes if etcd is unreachable
// and will not re-run until the elevator is restarted.
//
// Retry logic would be extremely helpful here.
func (e *Elevator) startPassengerWatcher() {

	go func(e *Elevator) {

		watcherOptions := client.WatcherOptions{AfterIndex: 0, Recursive: true}
		watcher := e.Etcd.KeysApi.Watcher("/wait/"+e.getKey(), &watcherOptions)
		for {
			r, err := watcher.Next(context.Background())
			if err != nil {
				fmt.Printf("Error from watcher.  Error: %v", err)
				return
			}
			// Need to process the passenger.
			// Start in new goroutine so we don't block the possible next one.
			go e.addNewWaitingPassengerFromNode(r.Node)
		}
	}(e)
}

// Start a watcher to deal with putting elevator into maintenance mode.
func (e *Elevator) startMaintenanceWatcher() {

	go func(e *Elevator) {

		watcherOptions := client.WatcherOptions{AfterIndex: 0, Recursive: true}
		watcher := e.Etcd.KeysApi.Watcher("/maintenance/"+e.getKey(), &watcherOptions)
		for {
			r, err := watcher.Next(context.Background())
			if err != nil {
				fmt.Printf("Error from watcher.  Error: %v", err)
				return
			}
			// Start in new goroutine so we don't block the possible next one.
			go e.updateMaintenanceModeFromNode(r.Node)
		}
	}(e)
}

// Moves the elevator into its next state.
// This state is determined by the current status of the Passengers and Waiting Passengers.
func (e *Elevator) move() {

	switch e.CurrentState {

	// If Elevator is idling, constantly check for passengers.
	case STATE_IDLE:

		if len(e.Waiting) > 0 {
			p := e.Waiting[0]
			if p.CurrentFloor > e.CurrentFloor {
				e.CurrentState = STATE_MOVING_UP
			} else if p.CurrentFloor < e.CurrentFloor {
				e.CurrentState = STATE_MOVING_DOWN
			} else {
				e.CurrentState = STATE_LOADING
			}
		}

	// If already moving up, check to see if passengers exist unloaded/loaded.
	case STATE_MOVING_UP:
		// Elevator is currently moving up.
		if (len(e.Passengers) > 0 || len(e.WaitingPassengers.Waiting) > 0) ||
			e.CurrentFloor < e.MaxFloor {

			e.CurrentState = STATE_MOVING_UP
			e.CurrentFloor++

			// We've "moved up" a floor.  Check to see if there are passengers to unload.
			if e.getUnloadPassengerCountForFloor() > 0 {
				e.CurrentState = STATE_UNLOADING
				return
			}

			// Also check to see if we can load more passengers.
			if e.getLoadPassengerCountForFloor() > 0 {
				e.CurrentState = STATE_LOADING
				return
			}

		} else {
			// Nothing left to do, so move self to IDLE.
			e.CurrentState = STATE_IDLE
		}

	// If already moving down, check to see if passengers exist.
	case STATE_MOVING_DOWN:
		if (len(e.Passengers) > 0 || len(e.WaitingPassengers.Waiting) > 0) ||
			e.CurrentFloor > e.MinFloor {

			e.CurrentState = STATE_MOVING_DOWN
			e.CurrentFloor--

			if e.getUnloadPassengerCountForFloor() > 0 {
				e.CurrentState = STATE_UNLOADING
				return
			}

			if e.getLoadPassengerCountForFloor() > 0 {
				e.CurrentState = STATE_LOADING
				return
			}

		} else {
			e.CurrentState = STATE_IDLE
		}

	case STATE_LOADING:
		// Elevator is currently loading passengers.
		e.loadPassengers()

		// Passengers are loaded, now decide which direction we were going in.
		if len(e.Passengers) > 0 {
			p := e.Passengers[0]
			if p.DestinationFloor > e.CurrentFloor {
				e.CurrentState = STATE_MOVING_UP
			} else {
				e.CurrentState = STATE_MOVING_DOWN
			}
		}

	case STATE_UNLOADING:
		// Elevator is currently unloading passengers.
		e.unloadPassengers()

		// IF current passengers exist, they're priority.
		// else if waiting passengers exist, then go to them.
		// else go idle.
		if len(e.Passengers) > 0 {

			p := e.Passengers[0]

			if p.DestinationFloor > e.CurrentFloor {

				e.CurrentState = STATE_MOVING_UP
				return
			} else if p.DestinationFloor < e.CurrentFloor {

				e.CurrentState = STATE_MOVING_DOWN
				return
			} else {

				e.CurrentState = STATE_IDLE
				return
			}
		}

		// Passengers are unloaded.  Now decide which direction we go in.
		if len(e.Waiting) > 0 {

			p := e.Waiting[0]

			if p.CurrentFloor > e.CurrentFloor {

				e.CurrentState = STATE_MOVING_UP
			} else if p.CurrentFloor < e.CurrentFloor {

				e.CurrentState = STATE_MOVING_DOWN
			} else {

				e.CurrentState = STATE_LOADING
			}
		} else {

			e.CurrentState = STATE_IDLE
		}

	case STATE_MAINTENANCE:
		// Doesn't respond to requests inputs.  Need to figure out a way to put an elevator in maintenance mode.
		if len(e.Waiting) > 0 {
			// TODO: Reschedule the passengers on a different lift.
		}

		if len(e.Passengers) > 0 {
			// Unload the passengers on the current floor.
			//
			// TODO:  Fix this.  Maintenance needs to be stored locally to
			// return to the maintenance mode upon unloading the passengers.
			// e.CurrentState = STATE_UNLOADING
			e.unloadPassengers()
			return
		}

		// This is bad implementation.  The state will change twice in the run loop.
		e.CurrentState = STATE_MAINTENANCE
	}
}

// Loads passengers into the elevator.
// Checks against the WaitingPassengers list to see if any passengers match
// If so, add them to the Passengers list and remove them from WaitingPassengers.
func (e *Elevator) loadPassengers() {
	if len(e.Waiting) > 0 {
		// If waiting passengers for this floor exist, load them into passengers.
		var waitingPassengers []*passenger.Passenger

		e.WaitingPassengers.Lock()

		for _, p := range e.Waiting {
			// Get the passengers waiting for this floor add all that are still waiting.
			if p.CurrentFloor != e.CurrentFloor {
				waitingPassengers = append(waitingPassengers, p)
			} else {
				e.addNewPassenger(p)
			}
		}

		e.Waiting = waitingPassengers
		e.WaitingPassengers.Unlock()
	}
}

// Unloads passengers from the elevator.
// Checks against the Passengers list to see if any passengers' destination floor matches the current floor.
// If yes, remove them from the passengers list.
func (e *Elevator) unloadPassengers() {

	if len(e.Passengers) > 0 {

		var passengers []*passenger.Passenger
		for _, p := range e.Passengers {
			// Get all passengers that are NOT on this current floor.
			// These will be the remaining passengers on the elevator.
			if p.DestinationFloor != e.CurrentFloor {
				passengers = append(passengers, p)
			}
		}
		e.Passengers = passengers
	}
}

// Returns a count of passengers to load for the current floor.
func (e *Elevator) getLoadPassengerCountForFloor() int {

	count := 0
	e.WaitingPassengers.Lock()

	for _, p := range e.WaitingPassengers.Waiting {

		if p.CurrentFloor == e.CurrentFloor {
			count++
		}
	}
	e.WaitingPassengers.Unlock()
	return count
}

// Returns a count of the passengers to unload for the current floor.
func (e *Elevator) getUnloadPassengerCountForFloor() int {

	count := 0
	for _, p := range e.Passengers {
		if p.DestinationFloor == e.CurrentFloor {
			count++
		}
	}

	return count
}

// Gets the key uniquely identifying the elevator in the cluster.
func (e *Elevator) getKey() string {
	return strconv.Itoa(e.GroupId) + "-" + strconv.Itoa(e.Id)
}

// Loads existing status from etcd cluster.
func (e *Elevator) loadExistingStatus() error {
	resp, err := e.KeysApi.Get(context.Background(), "elevators/"+e.getKey(), nil)
	if err != nil {
		fmt.Printf("Cannot get status.  Error: %+v\n", err)
		return err
	}

	var status ElevatorStatus
	if err = json.Unmarshal([]byte(resp.Node.Value), &status); err != nil {
		fmt.Printf("Cannott unmarshal json from etcd. Error: %v\n", err)
		return err
	}

	e.ElevatorStatus.Lock()
	// Copying this manually because setting the whole status causes a go vet error on TravisCI.
	// but not on my local machine running 1.6.2.
	e.ElevatorStatus.Id = status.Id
	e.ElevatorStatus.GroupId = status.GroupId
	e.ElevatorStatus.CurrentFloor = status.CurrentFloor
	e.ElevatorStatus.CurrentState = status.CurrentState
	e.ElevatorStatus.Passengers = status.Passengers
	e.ElevatorStatus.Waiting = status.Waiting
	e.ElevatorStatus.Unlock()
	return nil
}

// Saves the current state of the elevator.
// This is called on each full elevator Move.
func (e *Elevator) saveState() error {

	e.WaitingPassengers.Lock()
	data, err := json.Marshal(e.ElevatorStatus)
	e.WaitingPassengers.Unlock()

	// Once with /elevators/<key>
	// Current set to 3 seconds, but needs to change once the upper timer changes.
	setOptions := client.SetOptions{TTL: time.Duration(2) * time.Second}
	_, err = e.Etcd.KeysApi.Set(context.Background(), "elevator_status/"+e.getKey(), string(data), &setOptions)
	if err != nil {
		fmt.Printf("Error setting key to etcd.  Error: %s\n", err.Error())
		return err
	}

	_, err = e.Etcd.KeysApi.Set(context.Background(), "elevators/"+e.getKey(), string(data), nil)
	if err != nil {
		fmt.Printf("Error setting key to etcd.  Error: %s\n", err.Error())
		return err
	}

	return nil
}

func (e *Elevator) updateMaintenanceModeFromNode(node *client.Node) bool {
	maintMode, err := strconv.ParseBool(node.Value)
	if err != nil {
		fmt.Printf("Could not update maintenance mode.  Error: %+v\n", err)
		return false
	}

	if maintMode {
		e.CurrentState = STATE_MAINTENANCE
	} else {
		e.CurrentState = STATE_IDLE
	}

	return true
}

// Adds a new passenger to the WaitingPassengers list.
func (e *Elevator) addNewWaitingPassengerFromNode(node *client.Node) bool {
	var p passenger.Passenger
	err := json.Unmarshal([]byte(node.Value), &p)
	if err != nil {
		fmt.Printf("Could not unmarshal passenger.  Error: %+v\n", err)
		return false
	}

	ok := e.addNewWaitingPassenger(&p)
	if ok {
		e.saveState()
	}

	return ok
}

func (e *Elevator) addNewWaitingPassenger(p *passenger.Passenger) bool {
	e.WaitingPassengers.Lock()
	e.Waiting = append(e.Waiting, p)
	e.WaitingPassengers.Unlock()

	currentTarget := p.DestinationFloor
	for _, pass := range e.Waiting {
		if p.CurrentFloor > e.CurrentFloor && pass.DestinationFloor > currentTarget {
			currentTarget = pass.DestinationFloor
		} else if p.CurrentFloor < e.CurrentFloor && pass.DestinationFloor < currentTarget {
			currentTarget = pass.DestinationFloor
		}
	}
	e.CurrentTargetFloor = currentTarget

	return true
}

// Adds a new passenger to the Passengers list.
func (e *Elevator) addNewPassenger(p *passenger.Passenger) bool {

	e.Passengers = append(e.Passengers, p)
	return true
}

// Translates the const States into human-readable text and returns a string representation of
// the current state of the elevator.
func (e *Elevator) prettyPrintStatus() string {
	var currState string
	switch e.CurrentState {
	case STATE_IDLE:
		currState = "IDLE"
	case STATE_MOVING_UP:
		currState = "MOVING_UP"
	case STATE_MOVING_DOWN:
		currState = "MOVING_DOWN"
	case STATE_LOADING:
		currState = "LOADING"
	case STATE_UNLOADING:
		currState = "UNLOADING"
	case STATE_MAINTENANCE:
		currState = "MAINTENANCE"
	}

	e.WaitingPassengers.Lock()
	status := fmt.Sprintf("Elevator %d is on floor %d in state %s with %d passengers and %d waiting passengers.\n",
		e.DisplayId, e.CurrentFloor, currState, len(e.Passengers), len(e.WaitingPassengers.Waiting))
	e.WaitingPassengers.Unlock()

	return status
}
