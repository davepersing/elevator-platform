package passenger_test

import (
	"github.com/davepersing/elevator-platform/passenger"
	"testing"
)

func TestNewPassenger(t *testing.T) {
	p := passenger.Passenger{
		CurrentFloor:     1,
		DestinationFloor: 16,
	}

	if p.CurrentFloor != 1 || p.DestinationFloor != 16 {
		t.Errorf("Passenger expected to have CurrentFloor=1 and Destination=16, but got CurrentFloor=%d and Destination=%d",
			p.CurrentFloor, p.DestinationFloor)
	}
}
