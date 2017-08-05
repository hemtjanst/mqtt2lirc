package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"github.com/yosssi/gmq/mqtt/client"
	"io/ioutil"
	"os"
	"strings"
	"github.com/google/uuid"
	"time"
)

var (
	mqttAddressFlag  = flag.String("mqtt.address", "localhost:1883", "Address to MQTT endpoint")
	mqttUsernameFlag = flag.String("mqtt.username", "", "MQTT Username")
	mqttPasswordFlag = flag.String("mqtt.password", "", "MQTT Password")
	mqttTLSFlag      = flag.Bool("mqtt.tls", false, "Enable TLS")
	mqttCAFlag       = flag.String("mqtt.ca", "", "Path to CA certificate")
	mqttCertFlag     = flag.String("mqtt.cert", "", "Path to Client certificate")
	mqttKeyFlag      = flag.String("mqtt.key", "", "Path to Client certificate key")
)

type ClientConfig struct {
	WillTopic   string
	WillPayload string
	WillQoS     int
	WillRetain  bool
}

type Client struct {
	config       ClientConfig
	address      string
	username     string
	password     string
	cleanSession bool
	clientId     string
	tls          *tls.Config
	subscriptions []*client.SubscribeOptions
}

func NewPersistantMqtt(config ClientConfig) (mqtt *Client, err error) {

	if !flag.Parsed() {
		flag.Parse()
	}

	mqtt, err = setupClient(config)
	if err != nil {
		return
	}

	go mqtt.start()
	return
}

func envOrFlagStr(flagVal, envName, defVal string) (ret string) {
	ret = defVal

	if val, ok := os.LookupEnv(envName); ok {
		ret = val
	}
	if flagVal != defVal {
		ret = flagVal
	}
	return
}

func setupClient(config ClientConfig) (mqttClient *Client, err error) {
	useTls := false

	if val, ok := os.LookupEnv("MQTT_TLS"); ok {
		useTls = val != "0" && val != "" && strings.ToLower(val) != "false"
	}
	if *mqttTLSFlag {
		useTls = true
	}

	caPath := envOrFlagStr(*mqttCAFlag, "MQTT_CA_PATH", "")
	certPath := envOrFlagStr(*mqttCertFlag, "MQTT_CERT_PATH", "")
	keyPath := envOrFlagStr(*mqttKeyFlag, "MQTT_KEY_PATH", "")
	address := envOrFlagStr(*mqttAddressFlag, "MQTT_ADDRESS", "localhost:1883")
	username := envOrFlagStr(*mqttUsernameFlag, "MQTT_USERNAME", "")
	password := envOrFlagStr(*mqttPasswordFlag, "MQTT_PASSWORD", "")

	var tlsCfg *tls.Config
	if useTls {
		tlsCfg, err = setupTls(caPath, certPath, keyPath)
		if err != nil {
			return
		}
	}

	var rndUuid uuid.UUID
	rndUuid, err = uuid.NewRandom()
	if err != nil {
		return
	}

	mqttClient = &Client{
		config: config,
		tls:    tlsCfg,
		address: address,
		username: username,
		password: password,
		cleanSession: true,
		clientId: rndUuid.String(),
	}

	return
}

func setupTls(caPath, certPath, keyPath string) (*tls.Config, error) {
	tlsCfg := &tls.Config{}
	if caPath != "" {
		caPem, err := ioutil.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		tlsCfg.RootCAs = x509.NewCertPool()
		tlsCfg.RootCAs.AppendCertsFromPEM(caPem)
	}

	if certPath != "" && keyPath != "" {
		if keyPath == "" {
			return nil, errors.New("Certificate path specified, but key path missing")
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
		tlsCfg.BuildNameToCertificate()
	}

	return tlsCfg, nil
}

func (c *Client) mqttOpts() (opts *client.ConnectOptions) {
	opts = &client.ConnectOptions{
		Address:      c.address,
		UserName:     []byte(c.username),
		Password:     []byte(c.password),
		ClientID:     []byte(c.clientId),
		CleanSession: c.cleanSession,
		TLSConfig:    c.tls,
	}

	if c.config.WillTopic != "" {
		opts.WillMessage=[]byte(c.config.WillPayload)
		opts.WillTopic=    []byte(c.config.WillTopic)
		opts.WillQoS=      byte(uint8(c.config.WillQoS))
		opts.WillRetain=   c.config.WillRetain
	}
	return
}

func (c *Client) start() {
	c.cleanSession = true
	cli := client.New(&client.Options{
		ErrorHandler: func(err error) {

		},
	})

	for {
		err := cli.Connect(c.mqttOpts())
		if err != nil {

			time.Sleep()
		}
	}


	mqtt = client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {

		},
	})

	if err != nil {
		mqtt = nil
	}
	return
}

func (c *Client) Subscribe(options *client.SubscribeOptions) {

}