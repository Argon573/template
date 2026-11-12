package trip

import "errors"

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func NewCoordinates(latitude, longitude float64) (*Coordinates, error) {
	if latitude < -90 || latitude > 90 {
		return nil, errors.New("latitude must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return nil, errors.New("longitude must be between -180 and 180")
	}
	return &Coordinates{Latitude: latitude, Longitude: longitude}, nil
}
