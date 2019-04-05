package main

import (
  "os"
  "log"
  "fmt"
  "encoding/json"

  "github.com/joho/godotenv"
  "github.com/kuzemkon/aws-iot-device-sdk-go/device"
  "periph.io/x/periph/host"
  "periph.io/x/periph/conn/i2c/i2creg"
  "periph.io/x/periph/experimental/devices/mcp9808"
)

type Shadow struct {
  State struct {
    Reported struct {
      Temp int `json:"temp"`
    } `json:"reported"`
  } `json:"state"`
  Version int
}

func initEnv() error {
  home, err := os.UserHomeDir()
  if err != nil {
    return err
  }
  err = godotenv.Load(fmt.Sprintf("%s/.env", home))
  return err
}

func initLog() {
  logPath := os.Getenv("LOG_FILE_PATH")
  f, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
    log.Fatalf("error opening file: %v", err)
  }
  defer f.Close()

  log.SetOutput(f)
}

func initThing() (*device.Thing, error) {
  thingName := device.ThingName(os.Getenv("THING_NAME"))
  endpoint := os.Getenv("ENDPOINT")
  keyPair := device.KeyPair{
   PrivateKeyPath: os.Getenv("PRIVATE_KEY_PATH"),
   CertificatePath: os.Getenv("CERT_PATH"),
   CACertificatePath: os.Getenv("ROOT_CA_PATH"),
  }

  return device.NewThing(keyPair, endpoint, thingName)
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
  err := initEnv()
  if err != nil {
    panic(err)
  }

  initLog()

  // Initilize a new Thing
  thing, err := initThing()
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

