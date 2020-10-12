package main

import (
	"encoding/json"
	"fmt"
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
}

type control struct {
	Action      string
	Battery     int
	Linkquality int
}

func (m *mpzbc) mqttMessage(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Message: %s\n", msg.Payload())
	m.updateMPD()

	ctrl := control{}
	if err := json.Unmarshal(msg.Payload(), &ctrl); err != nil {
		fmt.Printf("Unmarshal error: %v", err)
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
}

func (m *mpzbc) connectMQTT() error {
	fmt.Println("Build MQTT Client")
	co := mqtt.NewClientOptions()
	co.AddBroker("tcp://" + m.mqttServer)
	co.SetClientID(fmt.Sprintf("mpzbc_%x", os.Getpid()))
	co.SetAutoReconnect(true)
	co.SetCleanSession(true)

	co.OnConnect = func(c mqtt.Client) {
		if token := c.Subscribe(m.mqttTopic, 0, m.mqttMessage); token.Wait() && token.Error() != nil {
			fmt.Printf("error during mqtt subscribe: %v\n", token.Error())
		}
	}

	client := mqtt.NewClient(co)
	fmt.Println("Connect to MQTT")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("error during mqtt connect: %v", token.Error())
	}

	return nil
}

func (m *mpzbc) updateMPD() error {
	status, err := m.mpdClient.Status()
	if err != nil {
		return fmt.Errorf("couldn't get MPD status: %v", err)
	}

	state, ok := status["state"]
	if !ok {
		return fmt.Errorf("no state in MPD status")
	}
	m.mpdStatus = state

	volume, ok := status["volume"]
	if !ok {
		m.mpdVolume = -1
	} else {
		m.mpdVolume, err = strconv.Atoi(volume)
		if err != nil {
			return fmt.Errorf("couldn't convert %s to integer: %v", volume, err)
		}
	}

	return nil
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

	m.mqttTopic = os.Getenv("MQTTTOPIC")
	if m.mqttTopic == "" {
		return fmt.Errorf("MQTTTOPIC is empty")
	}

	m.mpdServer = os.Getenv("MPDSERVER")
	if m.mpdServer == "" {
		return fmt.Errorf("MPDSERVER is empty")
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
		if err := m.updateMPD(); err != nil {
			return fmt.Errorf("couldn't update MPD state: %v", err)
		}
		fmt.Printf("state: %s\tvolume: %d\n", m.mpdStatus, m.mpdVolume)
		time.Sleep(1 * time.Second)
	}
}

func main() {
	client := mpzbc{}
	if err := client.run(); err != nil {
		fmt.Printf("error during client.run(): %v\n", err)
	}
}
