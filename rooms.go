package main

type Rooms struct {
	Rooms []Room `json:"foo"`
}

type Room struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}
