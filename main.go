package main

import(
    // "os/signal"
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
)

var log = logging.MustGetLogger("streambot-test")
var config * TestConfiguration

type Options struct {
    ConfigFilepath string `short:"c" long:"config" description:"File path of configuration file"`
}

// {
// 	"create_channel_throttle": 100,
// 	"subscribe_channel_throttle": 100,
// 	"fetch_subscriptions_throttle": 100
// }
type TestConfiguration struct {
	Host 						string 	`json:"host"`
	CreateChannelThrottle 		uint16 	`json:"create_channel_throttle"`
	SubscribeChannelThrottle 	uint16 	`json:"subscribe_channel_throttle"`
	SampleRate					float64 `json:"sample_rate"`
	GetSubscriptionThrottle 	uint16 `json:"get_subscription_throttle"`
}

func(config *TestConfiguration) Valid() bool {
	return config.Host != "" && config.SampleRate > 0
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
	runner := NewTestRunner(config)
	if runner == nil {
		log.Fatal("Unknown test runner `%s`", runner)
	}
	go runner.Start()
	// c := make(chan os.Signal, 1)                                       
	// signal.Notify(c, os.Interrupt)                                     
	// go func() {                                                        
	//   for sig := range c {                                             
	//     log.Debug("Captured %v, stopping API server..", sig)
	//     // runner.Stop()                                                
	//   }                                                                
	// }()
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
	ChannelIds 				[]string
	SubscribingChannelIds	[]string
	StatConn 				net.Conn
	Config					* TestConfiguration
	ChannelSampler 				* streambot.Sampler
	SubscriptionSampler 				* streambot.Sampler
}

func NewTestRunner(config *TestConfiguration) *TestRunner {
	r := new(TestRunner)
	r.API = streambot.NewAPI(config.Host)
	r.ChannelIds = []string{}
	r.SubscribingChannelIds = []string{}
	r.Config = config
	r.ChannelSampler = streambot.NewSampler(config.SampleRate)
	r.SubscriptionSampler = streambot.NewSampler(config.SampleRate)
	conn, err := net.Dial("udp", ":8125")
	if err != nil {
		log.Error("Error when instantiate UDP statting connection: %v", err)
	}
	r.StatConn = conn
	return r
}

type run func()

func Run(function run, throttle time.Duration) {
	go func(){
		dontStop := true
		go func(){
			for dontStop {
				function()
				time.Sleep(throttle)
			}
		}()
		// <- stop
		// dontStop = false
	}()
}

func (runner *TestRunner) StartChannelCreateBackgroundRunner() {
	Run(func() {
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
	}, time.Duration(int64(runner.Config.CreateChannelThrottle) * 1000 * 1000))
}

/**
 * TODO: Make API respond "Not Allowed" for FromChannel == ToChannel!
 */

func (runner *TestRunner) StartChannelSubscribeBackgroundRunner() {
	Run(func() {
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
	}, time.Duration(int64(runner.Config.SubscribeChannelThrottle) * 1000 * 1000))
}


func (runner *TestRunner) StartSubscriptionFetchBackgroundRunner() {
	Run(func() {
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
	}, time.Duration(int64(runner.Config.GetSubscriptionThrottle) * 1000 * 1000))
}

func (runner *TestRunner) Start() {
	log.Debug("Start")
	go func() {

	}()
	go runner.StartChannelCreateBackgroundRunner()
	go runner.StartChannelSubscribeBackgroundRunner()
	go runner.StartSubscriptionFetchBackgroundRunner()
}