[![Build Status](https://travis-ci.org/davepersing/elevator-platform.svg?branch=master)](https://travis-ci.org/davepersing/elevator-platform)[![GoDoc](https://godoc.org/github.com/davepersing/elevator-platform?status.svg)](https://godoc.org/github.com/davepersing/elevator-platform)

#Elevator Platform#
This is a coding exercise to outline a design for a scalable, fault-tolerant elevator system.

## Getting Started ##
1.  `git clone https://github.com/davepersing/elevator-platform`
2.  `cd elevator-platform`
3.  `make`

## Running the Application ##
1.  Start etcd.
2.  `make run` or `make build && elevator-platform <options>`

#### Command Line Options ####
-  `-groups=1` - Specifies the number of elevator groups to create.
-  `-elevators=2` - Specifies the number of elevators in a given elevator group.
-  `-capacity=16` - Specifies the maximum capacity of an elevator in persons.
-  `-bottom-floor=1` - Specifies the bottom floor the elevator is allowed to access.
-  `-top-floor=16` - Specifies the top floor the elevator is allowed to access.
-  `-etcd-url=http://localhost:2379` - Specifies the etcd cluster to connect to.

#### Interacting with the CLI ####
To add a new passenger:
Enter `new` and input the current floor and destination floor.

To exit:
Enter `exit`

To put an elevator into maintenance mode:
Enter `maint` with the group number, the elevator id, and true/false.


## Abstract Design ##
The system is comprised of two major pieces.

1.  ElevatorService is comprised of two parts:
  a.  Elevator - A state machine with accompanying functionality to operate itself.
  b.  HttpApi - Exposes a POST endpoint to schedule elevator calls.
2.  Etcd provides the backing data store and provides fault-tolerant, consistent data to drive the elevator system.


#### Security Model ####
Security would be provided by passing username/password over HTTPS to the scheduling server.  Upon successful authentication, the secured floor would be scheduled for the passenger.  Improvements to the scheduler would need to disallow additional passengers from boarding the elevator while in transit to secured floors.

Other possibilities include allowing anyone to go to the secured floor, but disallow exit until the passenger has entered correct credentials.


#### Etcd ####
Etcd was a logical choice as a backing data store due it's fault tolerance and consistency in the face of network partitions.  If part of the etcd cluster goes down, we can still have assurance in the integrity of the data stored there.

The system is dependent on etcd to maintain the current state of all elevators currently running.  If entire cluster goes down, the elevators will continue to run in their current state until they reach an IDLE state.  At that point, the elevator will be unable to create new scheduling requests or save it's current state.

Additionally, etcd stores the current statuses of each elevator in another key in order to provide a current state bound by a time-to-live (TTL).  This gives the system the ability to continue operating on a reduced number of elevators should a network partition occur between elevator systems.


#### Elevator Service ####
The ElevatorService is modeled after a [12Factor App](http://12factor.net/).  Configuration is via parameters passed into the application.

ElevatorService as a whole is intended to act as a microservice exposing an HTTP API while allow each individual Elevator to maintain internal state.

Any ElevatorService addressable by the load balancer is able to receive and process the passenger's call request.


#### Elevator ####
The Elevator inside of ElevatorService is a state machine with the following possible states:

```go
const (
  // Elevator states
  STATE_IDLE = iota
  STATE_MOVING_UP
  STATE_MOVING_DOWN
  STATE_MAINTENANCE
  STATE_LOADING
  STATE_UNLOADING
  STATE_ERROR
)
```

Each state is determined by the current state of the application on a timed run loop.  Upon creation, the elevator is created in STATE_IDLE with zero passengers and zero waiting passengers.

On each run loop iteration, the `move()` function is call to determine if a state transition is required.

On elevator initialization, the elevator makes two connections to the Etcd cluster:

1.  `GET /elevators/0-0` - Retrieves saved state of the elevator.
2.  `GET /wait/0-0` - Create a watcher to receive a notification when waiting passengers have been updated.

The current state is updated in Etcd during the following activities:

1.  The main run loop completes one full iteration.
2.  A waiting passenger has been added, the elevator has added that passenger to it's internal waiting passenger list.

The current state is updated for two keys during the `saveState()` call:

1.  `PUT /elevators/0-0` - The persisted state of the elevator.
2.  `PUT /elevator_status/0-0` - The current state of the elevator with a TTL.  This provides the scheduler with the ability to know which elevators can update their status.

On startup, `main.go` will initialize n elevators as defined in the command line flags.  Each elevator is assigned an Id via for loop and index.


#### HTTP API ####
The HTTP API exposes a single endpoint to allow any ElevatorService to schedule a passenger call with the system.

The HTTP API exposes one endpoint to the load balancer:

- `POST /elevator_call` - This takes a `Passenger` struct. The handler requests and elevator ID from the scheduler based on the current statuses of the elevators.

On startup, `main.go` will initialize the HTTP API with its own serve mux and port number.  The port number is determined by `8080 + i`


#### Scheduler ####
The scheduler maintains no internal state.  The receives a map of elevator statuses retrieved from etcd.  On a scheduler request, it iterate over all returned statuses to remove out-of-service elevators and elevators that may be in an error state.

After filtering the statuses, the elevator will schedule the passenger on an elevator by going through the following steps:
1.  Find elevators in `STATE_IDLE`
2.  Find elevators moving in same direction as the passenger, but have not passed their floor.
3.  Compare the values of the closest idle elevator and the elevator moving in the same direction as the passenger.  Choose whichever elevator is closest to the passenger.
3.  If steps 1 - 3 failed to find an elevator id, the scheduler will find the elevator with a target floor closest to the passenger's current floor.  The target floor is the highest or lowest floor the elevator will each during the current course of movement.

As a backup measure, if Step 3 fails to return an elevator, the scheduler will choose an elevator at random.  The worst case scenario will be the passenger waiting longer than normal for a ride.


#### (VERY) Simple Architectural Diagram ####

![Architecture Diagram](https://raw.githubusercontent.com/davepersing/elevator-platform/master/assets/HighLevelArch.jpg)

## Request Flow ##

![Request Flow](https://raw.githubusercontent.com/davepersing/elevator-platform/master/assets/RequestFlow.png)

1.  User request a new elevator by typing `new` while the console app is running.
2.  User inputs current floor and destination floor and presses Enter.
3.  Upon the request, the elevator that received the request will request all statuses from Etcd.
4.  After retrieving the statuses, the scheduler will decide which elevator the passenger should take.
5.  After the scheduling decision is made, the receiving elevator will set the target elevator's watched key
6.  Once the key is set, the target elevator will add the waiting passenger to it's internal waiting passenger list.
7.  On the next timer tick, the elevator will make an internal decision which direction to move to unload existing passengers and load the new passengers.


## Improvements ##

-  Improved handling of waiting passengers.  Currently, the system only handles a single passenger at a time.  This is dangerous due to the likely possibility to two passengers being scheduled at the same time.  One passenger could be overwritten and not picked up.
-  Improved scheduling for passengers that need a reschedule due to latency in the system.
-  Error handling for lost connections.  Currently, if the connection to etcd is lost, the watcher goroutine does not get restart.  Implement a exponential backoff reconnection scheme to try to restablish contact.
-  Implement capacity check in elevator scheduling to ensure elevator is not overburdened.
-  Separate CLI for new passenger and elevator status.
-  Admin mode to drive maintenance mode.
-  Maintenance mode currently immediately unloads passengers on the current floor, but does not change state to STATE_UNLOADING.  Maintenance needs to be stored in the status struct.
-  Improvements to how requests are scheduled.  Possibilities include moving scheduling HTTP API into its own server rather than being included with the elevator.
-  General code clean up.
-  Dashboard for elevator status.
-  Additional elements of an elevator as state machines.  Door, hydraulics, buttons, etc.
-  Improved documentation for godoc.
