package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"io"

	"github.com/kuzemkon/aws-iot-device-sdk-go/device"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/experimental/devices/mcp9808"
	"periph.io/x/periph/host"
)

type Shadow struct {
	State struct {
		Reported struct {
			Temp int `json:"temp"`
		} `json:"reported"`
	} `json:"state"`
	Version int
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

func main() {
	var (
		thingName  = flag.String("thing", "", "Set this to the AWS IoT thing name")
		endpoint   = flag.String("endpoint", "", "Set this to the AWS IoT endpoint")
		privateKey = flag.String("privatekey", "", "This must be a full path to the AWS IoT thing private key .pem.key file")
		cert       = flag.String("cert", "", "This must be a full path to the AWS IoT thing cert .pem.crt file")
		rootCA     = flag.String("rootca", "", "This must be a full path to the AWS IoT thing root-CA .crt file")
		logFile    = flag.String("logfile", "", "Set this to the full path for the log file")
	)
	flag.Parse()	
	
	if *thingName == "" || *endpoint == "" || *privateKey == "" || *cert == "" || *rootCA == "" || *logFile == "" {
		flag.PrintDefaults()
		panic("missing flag")
	}

	// Initilize logging
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// Initilize a new Thing
	keyPair := device.KeyPair{
		PrivateKeyPath:    *privateKey,
		CertificatePath:   *cert,
		CACertificatePath: *rootCA,
	}

	thing, err := device.NewThing(keyPair, *endpoint, device.ThingName(*thingName))
	if err != nil {
		panic(err)
	}

	// Subscribe to shadow
	shadowChan, err := thing.SubscribeForThingShadowChanges()
	if err != nil {
		panic(err)
	}

	// Read temperature from sensor
	data := readTemp()
	shadow := fmt.Sprintf(`{"state": {"reported": {"temp": %d}}}`, data)

	// Update thing shadow
	err = thing.UpdateThingShadow(device.Shadow(shadow))
	if err != nil {
		panic(err)
	}

	updatedShadow, ok := <-shadowChan
	if !ok {
		panic("Failed to read from shadow channel")
	}

	unmarshaledUpdatedShadow := &Shadow{}

	err = json.Unmarshal(updatedShadow, unmarshaledUpdatedShadow)
	if err != nil {
		panic(err)
	}

	log.Printf("Temp: %v Version: %v", unmarshaledUpdatedShadow.State.Reported.Temp, unmarshaledUpdatedShadow.Version)
}
