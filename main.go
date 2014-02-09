package main

import(
	stdlog "log"
    "os"
    "github.com/op/go-logging"
    "github.com/jessevdk/go-flags"
    "code.google.com/p/go-uuid/uuid"
    "net"
    "fmt"
    "encoding/json"
    "io/ioutil"
    "time"
    "streambot"
    "math/rand"
    "errors"
)

var log = logging.MustGetLogger("streambot-test")
var config * TestConfiguration

type Options struct {
    ConfigFilepath string `short:"c" long:"config" description:"File path of configuration file"`
}

type TestConfiguration struct {
	Hosts 						[]string 	`json:"hosts"`
	CreateChannelThrottle 		uint16 		`json:"create_channel_throttle"`
	SubscribeChannelThrottle 	uint16 		`json:"subscribe_channel_throttle"`
	SampleRate					float64 	`json:"sample_rate"`
	GetSubscriptionThrottle 	uint16  	`json:"get_subscription_throttle"`
	NumWorkers					float64 	`json:"num_workers"`
}

func(config *TestConfiguration) Valid() bool {
	return len(config.Hosts) > 0 && config.SampleRate > 0
}

func ReadConfig(file string) *TestConfiguration {
	var config TestConfiguration
	buf, err := ioutil.ReadFile(file)
    if err != nil {
		errMsgFormat := "Unexpected error on loading configuration from JSON file `%s`: %v"
		log.Fatal(fmt.Sprintf(errMsgFormat, file, err))
	}
    err = json.Unmarshal(buf, &config)
	if err != nil {
		errMsgFormat := "Unexpected error on loading configuration from JSON file `%s`: %v"
		log.Fatal(fmt.Sprintf(errMsgFormat, file, err))
	}
	if !config.Valid() {
		log.Fatalf("Invalid configuration: %v", string(buf))
	}
	return &config
}

func init() {
	var options Options
	var parser = flags.NewParser(&options, flags.Default)
    if _, err := parser.Parse(); err != nil {
    	fmt.Println(fmt.Sprintf("Error when parsing arguments: %v", err))
        os.Exit(1)
    }
    if options.ConfigFilepath == "" {
    	fmt.Println("Missing a valid configuration file specification argument. Usage: -c " +
    		"<config_file>")
    	os.Exit(1)	
    }
    config = ReadConfig(options.ConfigFilepath)
	// Customize the output format
    logging.SetFormatter(logging.MustStringFormatter("%{message}"))
    // Setup one stdout and one syslog backend.
    logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags/*|stdlog.Lshortfile*/)
    logBackend.Color = true
    syslogBackend, err := logging.NewSyslogBackend("")
    if err != nil {
        log.Fatal(err)
    }
    // Combine them both into one logging backend.
    logging.SetBackend(logBackend, syslogBackend)
    logging.SetLevel(logging.DEBUG, "streambot-test")
}

func main() {
	log.Debug(fmt.Sprintf("Main with config %v", config))
	runner, err := NewTestRunner(config)
	if err != nil {
		log.Fatal("Unexpected error when intiializing test runner: %v", err)
	}
	if runner == nil {
		log.Fatal("Unknown test runner `%s`", runner)
	}
	go runner.Start()
	for {
		time.Sleep(100 * time.Millisecond)
	}
}

type Subscription struct {
	From 	string
	To 		string
}

type TestRunner struct {
	API						* streambot.API
	Workers					[]*Worker
	ChannelIds 				[]string
	SubscribingChannelIds	[]string
	StatConn 				net.Conn
	Config					* TestConfiguration
	ChannelSampler 			* streambot.Sampler
	SubscriptionSampler 	* streambot.Sampler
}

func NewTestRunner(config *TestConfiguration) (r *TestRunner, err error) {
	api, err := streambot.NewAPI(config.Hosts)
	if err != nil {
		err = errors.New(fmt.Sprintf("Unexpected error when initializing test API client: %v", err))
		return
	}
	r = new(TestRunner)
	r.API = api
	r.ChannelIds = []string{}
	r.SubscribingChannelIds = []string{}
	r.Config = config
	r.ChannelSampler = streambot.NewSampler(config.SampleRate)
	r.SubscriptionSampler = streambot.NewSampler(config.SampleRate)
	r.Workers = []*Worker{}
	conn, err := net.Dial("udp", ":8125")
	if err != nil {
		log.Error("Error when instantiate UDP statting connection: %v", err)
	}
	r.StatConn = conn
	return
}


func NewWorker(runner *TestRunner) *Worker {
	return &Worker{runner}
}

type Worker struct {
	Runner *TestRunner
}

func(w *Worker) Work() {
	go func(){
		for {
			taskIdx := rand.Intn(3)
			var throttle uint16
			switch taskIdx {
			case 0: 
				w.Runner.CreateChannel()
				throttle = w.Runner.Config.CreateChannelThrottle
			case 1: 
				w.Runner.CreateSubscription()
				throttle = w.Runner.Config.SubscribeChannelThrottle
			case 2: 
				w.Runner.GetSubscriptions()
				throttle = w.Runner.Config.GetSubscriptionThrottle
			}
			time.Sleep(time.Duration(int64(throttle) * 1000 * 1000))
			
		}
	}()
}

func (runner *TestRunner) CreateChannel() {
	channelName := uuid.New()
	log.Debug(fmt.Sprintf("Create channel with name `%s`", channelName))
	// Synchronous web service call
	beforeDB := time.Now()	
	id, err := runner.API.CreateChannel(channelName)
	afterDB := time.Now()
	// Calculate database call duration and track in statter
	duration := afterDB.Sub(beforeDB)/time.Millisecond
	log.Debug("CreateChannel took %d", duration)
	statSigns := []string{}
	if err != nil {
		statSigns = append(statSigns, "test.TestRunner.errors.all")
		statSigns = append(statSigns, "test.TestRunner.errors.NewChannel")
		log.Fatalf("Error when create channel: %v", err)
	} else {
		statSigns = append(statSigns, "test.TestRunner.NewChannel")
		runner.ChannelSampler.SampleId(id)
	}
	for _, statSign := range statSigns {
		fmt.Fprintln(runner.StatConn, fmt.Sprintf("%s:1|c", statSign))
	}
}

/**
 * TODO: Make API respond "Not Allowed" for FromChannel == ToChannel!
 */

func (runner *TestRunner) CreateSubscription() {
	fromChannelId := runner.ChannelSampler.RandomSampledId()
	toChannelId := runner.ChannelSampler.RandomSampledId()
	if fromChannelId == "" || toChannelId == "" || fromChannelId == toChannelId {
		return;
	}
	log.Debug(fmt.Sprintf("Create subscription from channel `%s` to channel `%s`", fromChannelId, toChannelId))
	beforeDB := time.Now()	
	err := runner.API.CreateSubscription(fromChannelId, toChannelId)
	afterDB := time.Now()
	// Calculate database call duration and track in statter
	duration := afterDB.Sub(beforeDB)/time.Millisecond
	log.Debug("CreateSubscription took %d", duration)
	statSigns := []string{}
	if err != nil {
		statSigns = append(statSigns, "test.TestRunner.errors.all")
		statSigns = append(statSigns, "test.TestRunner.errors.NewSubscription")
	} else {
		statSigns = append(statSigns, "test.TestRunner.NewSubscription")
		runner.SubscriptionSampler.SampleId(fromChannelId)
	}
	for _, statSign := range statSigns {
		fmt.Fprintln(runner.StatConn, fmt.Sprintf("%s:1|c", statSign))
	}
}


func (runner *TestRunner) GetSubscriptions() {
	id := runner.SubscriptionSampler.RandomSampledId()
	if id == "" {
		return
	}
	fmt.Println(fmt.Sprintf("Get subscriptions channel ID is %s", id))
	channelIds, err := runner.API.GetSubscriptionsOfChannelWithId(id)
	log.Debug(fmt.Sprintf("Get subscriptions of channel `%s`", id))
	statSigns := []string{}
	if err != nil {
		log.Fatalf(fmt.Sprintf("Subscriptions of channel error `%v`", err))
		statSigns = append(statSigns, "test.TestRunner.errors.all")
		statSigns = append(statSigns, "test.TestRunner.errors.GetSubscriptions")
	} else {
		log.Debug(fmt.Sprintf("Subscriptions of channel `%v`", channelIds))
		statSigns = append(statSigns, "test.TestRunner.GetSubscriptions")
		if channelIds == nil {
			statSigns = append(statSigns, "test.TestRunner.errors.GetSubscriptionsEmpty")
		} else if len(channelIds) == 0 {
			statSigns = append(statSigns, "test.TestRunner.errors.GetSubscriptionsZero")
		} else {
			statSigns = append(statSigns, fmt.Sprintf("test.TestRunner.NumSubscriptions.%d", len(channelIds)))
		}
	}
	for _, statSign := range statSigns {
		fmt.Fprintln(runner.StatConn, fmt.Sprintf("%s:1|c", statSign))
	}
}

func (runner *TestRunner) Start() {
	for i := 1;  i<=int(runner.Config.NumWorkers); i++ {
		w := NewWorker(runner)
		w.Work()
		runner.Workers = append(runner.Workers, w)
		time.Sleep(time.Duration(int64(rand.Intn(1000)) * 1000 * 1000))
	}
}