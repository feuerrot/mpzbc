package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/fhs/gompd/mpd"
)

const volumedelta int = 5

type mpzbc struct {
	mqttClient mqtt.Client
	mqttServer string
	mqttTopic  string
	mpdClient  *mpd.Client
	mpdServer  string
	mpdStatus  string
	mpdVolume  int
	update     bool
}

type control struct {
	Action      string
	Battery     int
	Linkquality int
}

func (m *mpzbc) mqttMessage(client mqtt.Client, msg mqtt.Message) {
	m.printStatus()
	ctrl := control{}
	if err := json.Unmarshal(msg.Payload(), &ctrl); err != nil {
		log.Printf("Unmarshal error: %v", err)
	}

	switch ctrl.Action {
	case "play_pause":
		if m.mpdStatus == "play" {
			m.mpdClient.Pause(true)
		} else {
			m.mpdClient.Play(-1)
		}
	case "rotate_left":
		m.updateMPD()
		if m.mpdVolume != -1 {
			m.mpdClient.SetVolume(m.mpdVolume - volumedelta)
		}
	case "rotate_right":
		m.updateMPD()
		if m.mpdVolume != -1 {
			m.mpdClient.SetVolume(m.mpdVolume + volumedelta)
		}
	case "skip_backward":
		m.mpdClient.Previous()
	case "skip_forward":
		m.mpdClient.Next()
	}

	m.printStatus()
}

func (m *mpzbc) connectMQTT() error {
	log.Println("Build MQTT Client")
	co := mqtt.NewClientOptions()
	co.AddBroker("tcp://" + m.mqttServer)
	co.SetClientID(fmt.Sprintf("mpzbc_%x", os.Getpid()))
	co.SetAutoReconnect(true)
	co.SetCleanSession(true)

	co.OnConnect = func(c mqtt.Client) {
		if token := c.Subscribe(m.mqttTopic, 0, m.mqttMessage); token.Wait() && token.Error() != nil {
			log.Fatalf("error during mqtt subscribe: %v\n", token.Error())
		}
	}

	client := mqtt.NewClient(co)
	log.Println("Connect to MQTT")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("error during mqtt connect: %v", token.Error())
	}

	return nil
}

func (m *mpzbc) updateMPD() (bool, error) {
	update := false
	status, err := m.mpdClient.Status()
	if err != nil {
		return false, fmt.Errorf("couldn't get MPD status: %v", err)
	}

	state, ok := status["state"]
	if !ok {
		return false, fmt.Errorf("no state in MPD status")
	}
	if m.mpdStatus != state {
		update = true
	}
	m.mpdStatus = state

	volume, ok := status["volume"]
	if !ok {
		m.mpdVolume = -1
	} else {
		newVolume, err := strconv.Atoi(volume)
		if err != nil {
			return false, fmt.Errorf("couldn't convert %s to integer: %v", volume, err)
		}
		if m.mpdVolume != newVolume {
			update = true
		}
		m.mpdVolume = newVolume
	}

	return update, nil
}

func (m *mpzbc) connectMPD() error {
	mpdClient, err := mpd.Dial("tcp", m.mpdServer)
	if err != nil {
		return fmt.Errorf("error while connecting to %s: %v", m.mpdServer, err)
	}
	m.mpdClient = mpdClient

	return nil
}

func (m *mpzbc) getEnv() error {
	m.mqttServer = os.Getenv("MQTTSERVER")
	if m.mqttServer == "" {
		return fmt.Errorf("MQTTSERVER is empty")
	}
	log.Printf("MQTT Server\t%s\n", m.mqttServer)

	m.mqttTopic = os.Getenv("MQTTTOPIC")
	if m.mqttTopic == "" {
		return fmt.Errorf("MQTTTOPIC is empty")
	}
	log.Printf("MQTT Topic\t%s\n", m.mqttTopic)

	m.mpdServer = os.Getenv("MPDSERVER")
	if m.mpdServer == "" {
		return fmt.Errorf("MPDSERVER is empty")
	}
	log.Printf("MPD Server\t%s\n", m.mpdServer)

	return nil
}

func (m *mpzbc) printStatus() error {
	update, err := m.updateMPD()
	if err != nil {
		return fmt.Errorf("couldn't update MPD state: %v", err)
	}
	if update {
		log.Printf("state: %s\tvolume: %d\n", m.mpdStatus, m.mpdVolume)
	}

	return nil
}

func (m *mpzbc) run() error {
	if err := m.getEnv(); err != nil {
		return fmt.Errorf("couldn't get settings: %v", err)
	}
	if err := m.connectMPD(); err != nil {
		return fmt.Errorf("couldn't connect to MPD: %v", err)
	}
	if err := m.connectMQTT(); err != nil {
		return fmt.Errorf("couldn't connect to MQTT: %v", err)
	}
	for {
		if err := m.printStatus(); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC | log.Lmsgprefix)
	log.Println("mpzbc startup")
	client := mpzbc{}
	if err := client.run(); err != nil {
		log.Fatalf("error during client.run(): %v\n", err)
	}
}
