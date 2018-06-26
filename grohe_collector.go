package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const prefix = "grohe_"

var (
	upDesc               *prometheus.Desc
	tempDesc             *prometheus.Desc
	humDesc              *prometheus.Desc
	lastNotificationDesc *prometheus.Desc
)

func init() {
	upDesc = prometheus.NewDesc(prefix+"up", "Scrape was successful", nil, nil)
	l := []string{"house", "room", "appliance"}
	tempDesc = prometheus.NewDesc(prefix+"temp", "Air temperature (in degrees C)", l, nil)
	humDesc = prometheus.NewDesc(prefix+"humidity_percent", "Humidity", l, nil)
	l = append(l, "category")
	lastNotificationDesc = prometheus.NewDesc(prefix+"last_notification", "Timestmap of last notification per category", l, nil)
}

func get(path string, v interface{}) error {
	req, err := http.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Something went wrong")
	}
	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		return nil
	}

	err = json.Unmarshal([]byte("{\"foo\":"+string(body)+"}"), v)
	if err != nil {
		return nil
	}
	return nil
}

func getLocations() (Locations, error) {
	l := Locations{}
	err := get(base+locationsPath, &l)
	return l, err
}

func getRooms(location int) (Rooms, error) {
	r := Rooms{}
	path := fmt.Sprintf(roomsPath, location)
	err := get(base+path, &r)
	return r, err
}

func getAppliances(location, room int) (Appliances, error) {
	a := Appliances{}
	path := fmt.Sprintf(appliancesPath, location, room)
	err := get(base+path, &a)
	return a, err
}

func getApplianceData(location, room int, appliance string) (ApplianceData, error) {
	a := ApplianceData{}
	path := fmt.Sprintf(applianceDataPath, location, room, appliance)
	err := get(base+path, &a)
	return a, err
}

func getApplianceNotifications(location, room int, appliance string) (ApplianceNotifications, error) {
	a := ApplianceNotifications{}
	path := fmt.Sprintf(applianceNotificationsPath, location, room, appliance)
	err := get(base+path, &a)
	return a, err
}

type groheCollector struct {
}

func (c groheCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- tempDesc
	ch <- humDesc
	ch <- lastNotificationDesc
}

func (c groheCollector) Collect(ch chan<- prometheus.Metric) {
	locations, err := getLocations()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error locations data", err)
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1)
	for _, location := range locations.Locations {

		rooms, err2 := getRooms(location.ID)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "can't get rooms from %s\n", location.Name)
			continue
		}
		for _, room := range rooms.Rooms {
			appliances, err3 := getAppliances(location.ID, room.ID)
			if err3 != nil {
				fmt.Fprintf(os.Stderr, "can't get appliances from %s\n", room.Name)
				continue
			}
			for _, appliance := range appliances.Appliances {
				data, err4 := getApplianceData(location.ID, room.ID, appliance.ID)
				if err4 != nil {
					fmt.Fprintf(os.Stderr, "can't get data from %s\n", appliance.ID)
					continue
				}
				ch <- prometheus.MustNewConstMetric(tempDesc, prometheus.GaugeValue, float64(data.Data.Measurement[0].Temperature), location.Name, room.Name, appliance.Name)
				ch <- prometheus.MustNewConstMetric(humDesc, prometheus.GaugeValue, float64(data.Data.Measurement[0].Humidity), location.Name, room.Name, appliance.Name)
			}
			for _, appliance := range appliances.Appliances {
				notifications, err4 := getApplianceNotifications(location.ID, room.ID, appliance.ID)
				if err4 != nil {
					fmt.Fprintf(os.Stderr, "can't get notifications from %s\n", appliance.ID)
					continue
				}
				n := make(map[int]time.Time)
				for _, notification := range notifications.ApplianceNotification {
					t, err := time.Parse(time.RFC3339, notification.Timestamp)
					if err != nil {
						continue
					}
					_, ok := n[notification.Category]
					if ok && n[notification.Category].After(t) == true {
						continue
					}
					n[notification.Category] = t
				}
				for category, time := range n {
					ch <- prometheus.MustNewConstMetric(lastNotificationDesc, prometheus.GaugeValue, float64(time.Unix()), location.Name, room.Name, appliance.ID, strconv.Itoa(category))
				}
			}
		}
	}
}
