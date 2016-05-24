package passenger

// Defines a passenger.
type Passenger struct {
	//	Weight           uint // TODO:  Calculate capacity based on weight?
	CurrentFloor     int `json:"currentFloor"`     // The floor the passenger is currently on.
	DestinationFloor int `json:"destinationFloor"` // The floor the passenger wants to go to.
}
