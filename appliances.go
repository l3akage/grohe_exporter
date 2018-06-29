package main

type Appliances struct {
	Appliances []Appliance `json:"foo"`
}

type Appliance struct {
	ID   string `json:"appliance_id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

type ApplianceNotifications struct {
	ApplianceNotification []ApplianceNotification `json:"foo"`
}

type ApplianceNotification struct {
	ID        string `json:"id"`
	Category  int    `json:"category"`
	Type      int    `json:"type"`
	Timestamp string `json:"timestamp"`
}
