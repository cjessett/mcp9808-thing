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
  home, err := os.UserHomeDir()
  if err != nil {
    panic(err)
  }
  err = godotenv.Load(fmt.Sprintf("%s/.env", home))
  if err != nil {
    panic(err)
  }

  f, err := os.OpenFile(os.Getenv("LOG_FILE_PATH"), os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
    log.Fatalf("error opening file: %v", err)
  }
  defer f.Close()

  log.SetOutput(f)
  
  thingName := device.ThingName(os.Getenv("THING_NAME"))
  endpoint := os.Getenv("ENDPOINT")
  keyPair := device.KeyPair{
   PrivateKeyPath: os.Getenv("PRIVATE_KEY_PATH"),
   CertificatePath: os.Getenv("CERT_PATH"),
   CACertificatePath: os.Getenv("ROOT_CA_PATH"),
  }

  thing, err := device.NewThing(keyPair, endpoint, thingName)
  if err != nil {
    panic(err)
  }
  
  shadowChan, err := thing.SubscribeForThingShadowChanges()
  if err != nil {
    panic(err)
  }

  data := readTemp()
  shadow := fmt.Sprintf(`{"state": {"reported": {"temp": %d}}}`, data)
  
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

