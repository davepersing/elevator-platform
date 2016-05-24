package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/davepersing/elevator-platform/elevator"
	"github.com/davepersing/elevator-platform/elevator_service"
	"github.com/davepersing/elevator-platform/etcd"
	"github.com/davepersing/elevator-platform/http_api"
	"github.com/davepersing/elevator-platform/passenger"
	"github.com/davepersing/elevator-platform/util"
)

type StartupParams struct {
	ElevatorGroups int    // Number of elevator groups to start with n elevators.
	ElevatorCount  int    // Number of elevators to start up
	MaxFloor       int    // The maximum floor the elevator is able to access.
	MinFloor       int    // The minimum floor the elevator is able to access.
	MaxCapacity    int    // The maximum number of persons allowed in an elevator at any given time.
	EtcdUrl        string // The URL to the etcd cluster.
}

// 1.  `-groups=1` - Specifies the number of elevator groups to create.
// 2.  `-elevators=2` - Specifies the number of elevators in a given elevator group.
// 3.  `-capacity=16` - Specifies the maximum capacity of an elevator in persons.
// 4.  `-bottom-floor=1` - Specifies the bottom floor the elevator is allowed to access.
// 5.  `-top-floor=16` - Specifies the top floor the elevator is allowed to access.

// Starts the application.
func main() {
	initHTTPDefaults()

	fmt.Println("Enter 'new' to add a new passenger or 'exit' to leave.")

	var groupCount = flag.Int("groups", 1, "Number of elevator groups to start with n elevators.")
	var elevatorCount = flag.Int("elevators", 2, "Number of elevators to start within the group.")
	var bottomFloor = flag.Int("bottom-floor", 1, "The bottom floor the elevator can access.")
	var topFloor = flag.Int("top-floor", 16, "The top floor the elevator can access.")
	var maxCapacity = flag.Int("capacity", 16, "The maximum number of persons an elevator can carry at one time.")
	var etcdUrl = flag.String("etcd-url", "http://localhost:2379", "The url to the etcd cluster.  e.g. http://localhost:2379")

	flag.Parse()

	startupParams := StartupParams{
		ElevatorGroups: *groupCount,
		ElevatorCount:  *elevatorCount,
		MaxFloor:       *topFloor,
		MinFloor:       *bottomFloor,
		MaxCapacity:    *maxCapacity,
		EtcdUrl:        *etcdUrl,
	}

	knownNodes := startupParams.initElevators()

	// Init the scheduler in a goroutine.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "exit":
			fmt.Println("Should exit...")
			break
		case "new":
			startupParams.processNewPassenger(knownNodes)
			break
		}
	}
}

// Processes a new passenger from user input.
// Right now, not the best UX.  The status goroutines overwrite when trying to enter.
// Need to figure out a way to pause the output while entering new data.
func (s *StartupParams) processNewPassenger(knownNodes map[int]string) {
	fmt.Println("Enter your current floor: ")
	currFloor, err := s.getFloorInput()
	if err != nil {
		fmt.Printf("Error processing current floor: %s\n", err.Error())
		return
	}

	fmt.Println("Enter your destination floor: ")
	destFloor, err := s.getFloorInput()
	if err != nil {
		fmt.Printf("Error processing destination floor: %s\n", err.Error())
		return
	}

	// more validation that this is a valid request.
	if currFloor == destFloor {
		// Passenger is on the same floor, so dont' do anything.
		fmt.Println("You're already on the same floor!")
		return
	}

	// Everything checks out, so make a request to one of the elevators.
	// This is "load balancing".  Could implement round-robin, but this will do for now.
	randomIndex := util.GetRandomIndex(len(knownNodes) - 1)
	port := knownNodes[randomIndex]
	elevatorId, err := util.SendPassengerPost(port, currFloor, destFloor)
	if err != nil {
		fmt.Printf("Could not send request.  Error: %s\n", err.Error())
		return
	}

	fmt.Printf("Take Elevator %d\n", elevatorId)
}

// Gets the passenger input from Stdin
func (s *StartupParams) getFloorInput() (int, error) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		parsed, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return -1, err
		}

		if parsed <= 0 {
			return -1, errors.New("Floor must be greater than 0!")
		}

		if parsed > s.MaxFloor {
			return -1, errors.New("Floor is too high!  The top floor is " + strconv.Itoa(s.MaxFloor))
		}

		return parsed, nil
	}
	return -1, nil
}

// Initializes the elevators.
func (s StartupParams) initElevators() map[int]string {

	// TODO:  Create elevator Groups.
	//
	// Generate the cluster info based on the number of elevators.
	knownNodes := make(map[int]string)
	services := make(map[int]*elevator_service.ElevatorService)

	for i := 0; i < s.ElevatorCount; i++ {
		knownNodes[i] = ":" + strconv.Itoa(8080+i)

		es := &elevator_service.ElevatorService{
			HttpApi: &http_api.HttpApi{
				Hostname: "",
				Port:     knownNodes[i],
				Etcd:     &etcd.Etcd{Url: s.EtcdUrl}, // Shouldn't have to pass mulitple refs around.
			},
			Elevator: &elevator.Elevator{
				MaxFloor:    s.MaxFloor,
				MinFloor:    s.MinFloor,
				MaxCapacity: s.MaxCapacity,
				Etcd:        &etcd.Etcd{Url: s.EtcdUrl},
				ElevatorStatus: elevator.ElevatorStatus{
					DisplayId:         i + 1,
					GroupId:           0, // This is zero because only dealing with a single bank
					Id:                i, // Make this not 0-based.
					CurrentFloor:      s.MinFloor,
					CurrentState:      elevator.STATE_IDLE,
					WaitingPassengers: elevator.WaitingPassengers{Waiting: make([]*passenger.Passenger, 0)},
					Passengers:        make([]*passenger.Passenger, 0),
				},
			},
		}
		es.Init()

		services[i] = es
	}

	return knownNodes
}

// Initializes default parameters for HTTP requests using http.DefaultClient.
func initHTTPDefaults() {
	http.DefaultClient = &http.Client{
		Timeout: 100 * time.Millisecond,
	}
}
