package main

type Locations struct {
	Locations []Location `json:"foo"`
}

type Location struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}
