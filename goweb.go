package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
    "os/signal"
	"runtime"
	"io/ioutil"
    "strconv"
    "strings"    
    "syscall"
	"goweb/packages/route"
	"goweb/packages/database"
	"goweb/packages/email"
	"goweb/packages/jsonconfig"
	"goweb/packages/pushover"
	"goweb/packages/recaptcha"
	"goweb/packages/server"
	"goweb/packages/session"
	"goweb/packages/view"
	"goweb/packages/view/plugin"
)

var PIDFile = "/tmp/goblog.pid"

func savePID(pid int) {
    file, err := os.Create(PIDFile)
    if err != nil {
        log.Printf("Unable to create pid file : %v\n", err)
        os.Exit(1)
    }
    defer file.Close()
    _, err = file.WriteString(strconv.Itoa(pid))
    if err != nil {
        log.Printf("Unable to create pid file : %v\n", err)
        os.Exit(1)
    }
    file.Sync()
}

// *****************************************************************************
// Application Logic
// *****************************************************************************

func init() {
	// Verbose logging with file name and line number
	log.SetFlags(log.Lshortfile)

	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	if len(os.Args) != 2 {
        log.Printf("Usage : %s [start|stop] \n ", os.Args[0])
        os.Exit(0)
    }
    if strings.ToLower(os.Args[1]) == "main" {
        ch := make(chan os.Signal, 1)
        signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)
        go func() {
            signalType := <- ch
            signal.Stop(ch)
            log.Println("Exit command received. Exiting...")
            log.Println("Received signal type : ", signalType)
            os.Remove(PIDFile)
            os.Exit(0)
        }()
        rootDir, err := os.Getwd()
        if err != nil {
            log.Fatal(err)
        }
		// Load the configuration file
		jsonconfig.Load(rootDir + string(os.PathSeparator) + "config"+string(os.PathSeparator)+"config.json", config)

		// Configure the session cookie store
		session.Configure(config.Session)

		// Connect to database
		database.Connect(config.Database)

		// Configure the SMTP server
		email.Configure(config.Email)

		// Configure the Google reCAPTCHA prior to loading view plugins
		recaptcha.Configure(config.Recaptcha)

		// Configure Pushover
		pushover.Configure(config.Pushover)

		// Setup the views
		view.Configure(config.View)
		view.LoadTemplates(config.Template.Root, config.Template.Children)
		view.LoadPlugins(plugin.TemplateFuncMap(config.View))

		// Start the listener
		server.Run(route.LoadHTTP(), route.LoadHTTPS(), config.Server)
	}
	if strings.ToLower(os.Args[1]) == "start" {
        if _, err := os.Stat(PIDFile); err == nil {
            log.Println("Already running or pid file exist.")
            os.Exit(1)
        }
        cmd := exec.Command(os.Args[0],"main")
        cmd.Start()
        log.Println("Daemon process ID is : ", cmd.Process.Pid)
        savePID(cmd.Process.Pid)
        os.Exit(0)
    }
    if strings.ToLower(os.Args[1]) == "stop" {
        if _, err := os.Stat(PIDFile); err == nil {
            data, err := ioutil.ReadFile(PIDFile)
            if err != nil {
                log.Println("Not running")
                os.Exit(1)
            }
            ProcessID, err := strconv.Atoi(string(data))
            if err != nil {
                log.Println("Unable to read and parse process id found in ", PIDFile)
                os.Exit(1)
            }
            process, err := os.FindProcess(ProcessID)
            if err != nil {
                log.Printf("Unable to find process ID [%v] with error %v \n", ProcessID, err)
                os.Exit(1)
            }

            os.Remove(PIDFile)
            log.Printf("Kill pricess ID [%v] now.\n", ProcessID)
            
            err = process.Kill()
            if err != nil {
                log.Printf("Unable to kill process ID [%v] with error %v \n", ProcessID, err)
                os.Exit(1)
            } else {
                log.Printf("Kill process ID [%v]\n", ProcessID)
                os.Exit(0)
            }
        } else {
            log.Println("Not running.")
            os.Exit(1)
        }
    } else {
        log.Printf("Unknown command : %v\n", os.Args[1])
        os.Exit(1)
    }
}

// *****************************************************************************
// Application Settings
// *****************************************************************************

// config the settings variable
var config = &configuration{}

// configuration contains the application settings
type configuration struct {
	Database  database.MySQLInfo      `json:"Database"`
	Email     email.SMTPInfo          `json:"Email"`
	Recaptcha recaptcha.RecaptchaInfo `json:"Recaptcha"`
	Pushover  pushover.PushoverInfo   `json:"Pushover"`
	Server    server.Server           `json:"Server"`
	Session   session.Session         `json:"Session"`
	Template  view.Template           `json:"Template"`
	View      view.View               `json:"View"`
}

// ParseJSON unmarshals bytes to structs
func (c *configuration) ParseJSON(b []byte) error {
	return json.Unmarshal(b, &c)
}
