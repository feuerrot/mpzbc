# mpzbc - music player zigbee client
[zigbee2mqtt](https://www.zigbee2mqtt.io) + [IKEA E1744 SYMFONISK sound controller](https://www.zigbee2mqtt.io/devices/E1744.html) + mpzbc == wireless mpd remote

## Usage
```
go get github.com/feuerrot/mpzbc
MQTTSERVER=mqtthost:1883 MQTTTOPIC=zigbee2mqtt/friendly_e1744_name MPDSERVER=mpdhost:6600 mpzbc
```
