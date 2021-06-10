package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const (
	version                    string = "0.1"
	base                       string = "https://idp2-apigw.cloud.grohe.com"
	login                      string = "/v2/iot/auth/users/login"
	locationsPath              string = "/v2/iot/locations"
	roomsPath                  string = "/v2/iot/locations/%d/rooms"
	appliancesPath             string = "/v2/iot/locations/%d/rooms/%d/appliances"
	applianceDataPath          string = "/v2/iot/locations/%d/rooms/%d/appliances/%s/data"
	applianceNotificationsPath string = "/v2/iot/locations/%d/rooms/%d/appliances/%s/notifications"
)

var (
	showVersion   = flag.Bool("version", false, "Print version information.")
	listenAddress = flag.String("listen-address", ":9441", "Address on which to expose metrics.")
	metricsPath   = flag.String("path", "/metrics", "Path under which to expose metrics.")
	username      = flag.String("username", "", "Username")
	password      = flag.String("password", "", "Password")

	token string
)

func init() {
	flag.Usage = func() {
		fmt.Println("Usage: grohe_exporter [ ... ]\n\nParameters:")
		fmt.Println()
		flag.PrintDefaults()
	}
}

func getToken() (string, error) {
	t := Token{}
	var jsonStr = []byte(`{"username":"` + *username + `", "password":"` + *password + `"}`)
	req, err := http.NewRequest("POST", base+login, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("Wrong credentials")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(body), &t)
	if err != nil {
		return "", err
	}
	return t.Token, nil
}

func main() {
	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}
	var err error
	token, err = getToken()
	if err != nil {
		log.Fatal(err)
	}

	startServer()
}

func printVersion() {
	fmt.Println("grohe_exporter")
	fmt.Printf("Version: %s\n", version)
}

func startServer() {
	log.Infof("Starting grohe exporter (Version: %s)\n", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>grohe Exporter (Version ` + version + `)</title></head>
			<body>
			<h1>grohe Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			<h2>More information:</h2>
			<p><a href="https://github.com/l3akage/grohe_exporter">github.com/l3akage/grohe_exporter</a></p>
			</body>
			</html>`))
	})
	http.HandleFunc(*metricsPath, handleMetricsRequest)

	log.Infof("Listening for %s on %s\n", *metricsPath, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func handleMetricsRequest(w http.ResponseWriter, r *http.Request) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(&groheCollector{})

	promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      log.NewErrorLogger(),
		ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, r)
}
