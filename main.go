package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kuzemkon/aws-iot-device-sdk-go/device"
	"github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/experimental/devices/mcp9808"
	"periph.io/x/periph/host"
)

var thingName string
var endpoint string
var privateKey string
var cert string
var rootCA string
var logFile string
var log = logrus.New()

func init() {
	flag.StringVar(&thingName, "thing", "", "Set this to the AWS IoT thing name")
	flag.StringVar(&endpoint, "endpoint", "", "Set this to the AWS IoT endpoint")
	flag.StringVar(&privateKey, "privatekey", "", "This must be a full path to the AWS IoT thing private key .pem.key file")
	flag.StringVar(&cert, "cert", "", "This must be a full path to the AWS IoT thing cert .pem.crt file")
	flag.StringVar(&rootCA, "rootca", "", "This must be a full path to the AWS IoT thing root-CA .crt file")
	flag.StringVar(&logFile, "logfile", "", "Set this to the full path for the log file")
	flag.Parse()

	if thingName == "" || endpoint == "" || privateKey == "" || cert == "" || rootCA == "" || logFile == "" {
		flag.PrintDefaults()
		log.Panic("missing required flag")
	}

	// Initilize logging
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Info("error opening file: %v", err)
	}

	mw := io.MultiWriter(os.Stdout, file)

	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(mw)
}

func readTemp() int {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Open default I²C bus.
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("failed to open I²C: %v", err)
	}
	defer bus.Close()

	// Create a new temperature sensor.
	sensor, err := mcp9808.New(bus, &mcp9808.DefaultOpts)
	if err != nil {
		log.Fatalln(err)
	}

	// Read values from sensor.
	measurement, err := sensor.SenseTemp()

	if err != nil {
		log.Fatalln(err)
	}

	return int(measurement.Fahrenheit())
}

type Shadow struct {
	State struct {
		Reported struct {
			Temp int `json:"temp"`
		} `json:"reported"`
	} `json:"state"`
	Version int
}

func main() {
	// Initilize a new Thing
	keyPair := device.KeyPair{
		PrivateKeyPath:    privateKey,
		CertificatePath:   cert,
		CACertificatePath: rootCA,
	}

	thing, err := device.NewThing(keyPair, endpoint, device.ThingName(thingName))
	if err != nil {
		log.Panic(err)
	}

	// Subscribe to shadow
	shadowChan, err := thing.SubscribeForThingShadowChanges()
	if err != nil {
		log.Panic(err)
	}

	// Read temperature from sensor
	data := readTemp()
	shadow := fmt.Sprintf(`{"state": {"reported": {"temp": %d}}}`, data)

	// Update thing shadow
	err = thing.UpdateThingShadow(device.Shadow(shadow))
	if err != nil {
		log.Panic(err)
	}

	updatedShadow, ok := <-shadowChan
	if !ok {
		log.Panic("Failed to read from shadow channel")
	}

	unmarshaledUpdatedShadow := &Shadow{}

	err = json.Unmarshal(updatedShadow, unmarshaledUpdatedShadow)
	if err != nil {
		log.Panic(err)
	}

	log.WithFields(logrus.Fields{
		"temperature": unmarshaledUpdatedShadow.State.Reported.Temp,
		"version":     unmarshaledUpdatedShadow.Version,
	}).Info("Updated Shadow")
}
