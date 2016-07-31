// Copyright (c) 2016 John Beil.
// Use of this source code is governed by the MIT License.
// The MIT license can be found in the LICENSE file.

// BuoyBot 1.5
// Obtains latest observation for NBDC Stations
// Saves observation to database
// Tweets observation
// See README.md for setup information

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	_ "github.com/lib/pq"
)

// URL format for SF Buoy Observations
const noaaURLFmt = "http://www.ndbc.noaa.gov/data/realtime2/%s.txt"

// Observation struct stores buoy observation data
type Observation struct {
	Date                  time.Time
	SignificantWaveHeight float64
	DominantWavePeriod    int
	AveragePeriod         float64
	MeanWaveDirection     string
	WaterTemperature      float64
}

// Config struct stores Twitter and Database credentials and buoy ID
type Config struct {
	UserName         string `json:"UserName"`
	ConsumerKey      string `json:"ConsumerKey"`
	ConsumerSecret   string `json:"ConsumerSecret"`
	Token            string `json:"Token"`
	TokenSecret      string `json:"TokenSecret"`
	DatabaseFile     string `json:"DatabaseFile"`
	BuoyId           string `json:"BuoyId"`
}

// Variable for database
var db *sql.DB

func main() {
	fmt.Println("Starting BuoyBot...")

	// Load configuration
	config := Config{}
	loadConfig(&config)

	// Load database
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error: Could not establish connection with the database.", err)
	}

	// Get latest observation and store in struct
	var observation Observation
	observation = getObservation(config.BuoyId)

	// Save latest observation in database
	saveObservation(observation)

	// Format observation given Observation
	observationOutput := formatObservation(observation)

	// Tweet observation at 0000, 0600, 0800, 1000, 1200, 1400, 1600|| 1800 PST
	var loc *time.Location
	loc, err = time.LoadLocation("US/Pacific")
	if err != nil {
		log.Fatal("Error loading location:", err)
	}

	t := time.Now().In(loc)
	fmt.Println(t)
	if t.Hour() == 5 || t.Hour() == 7 || t.Hour() == 9 || t.Hour() == 11 || t.Hour() == 13 || t.Hour() == 16 || t.Hour() == 18 || t.Hour() == 20 {
		tweetCurrent(config, observationOutput)
	} else {
		fmt.Println("Not at update interval - not tweeting.")
		fmt.Println(observationOutput)
	}

	// Shutdown BuoyBot
	fmt.Println("Exiting BuoyBot...")
}

// Fetches and parses latest NBDC observation and returns data in Observation struct
func getObservation(buoyId string) Observation {
	var noaaURL = fmt.Sprintf(noaaURLFmt, buoyId)
	observationRaw := getDataFromURL(noaaURL)
	observationData := parseData(observationRaw)
	return observationData
}

// Given Observation struct, saves most recent observation in database
func saveObservation(o Observation) {
	_, err := db.Exec("INSERT INTO observations(observationtime, significantwaveheight, dominantwaveperiod, averageperiod, meanwavedirection, watertemperature) VALUES($1, $2, $3, $4, $5, $6)", o.Date, o.SignificantWaveHeight, o.DominantWavePeriod, o.AveragePeriod, o.MeanWaveDirection, o.WaterTemperature)
	if err != nil {
		log.Fatal("Error saving observation:", err)
	}
}

// Given config and observation, tweets latest update
func tweetCurrent(config Config, o string) {
	fmt.Println("Preparing to tweet observation...")
	var api *anaconda.TwitterApi
	api = anaconda.NewTwitterApi(config.Token, config.TokenSecret)
	anaconda.SetConsumerKey(config.ConsumerKey)
	anaconda.SetConsumerSecret(config.ConsumerSecret)
	tweet, err := api.PostTweet(o, nil)
	if err != nil {
		fmt.Println("update error:", err)
	} else {
		fmt.Println("Tweet posted:")
		fmt.Println(tweet.Text)
	}
}

// Given URL, returns raw data with recent observations from NBDC
func getDataFromURL(url string) (body []byte) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("ioutil error reading resp.Body:", err)
	}
	// fmt.Println("Status:", resp.Status)
	return
}

// Given path to config.js file, loads credentials
func loadConfig(config *Config) {
	// Load path to config from CONFIGPATH environment variable
	configpath := os.Getenv("CONFIGPATH")
	if configpath != "" {
		file, _ := os.Open(configpath)
		decoder := json.NewDecoder(file)
		err := decoder.Decode(&config)
		if err != nil {
			log.Fatal("Error loading config.json:", err)
		}
	} else {
		log.Fatal("CONFIGPATH environment variable not specified")
	}
}

// Given raw data, parses latest observation and returns Observation struct
func parseData(d []byte) Observation {
	// Each line contains 19 data points
	// Headers are in the first two lines
	// Latest observation data is in the third line
	// Other lines are not needed

	// Extracts relevant data into variable for processing
	var data = string(d[188:281])
	// Convert most recent observation into array of strings
	datafield := strings.Fields(data)

	// Convert wave height from meters to feet
	waveheightmeters, _ := strconv.ParseFloat(datafield[8], 64)
	waveheightfeet := waveheightmeters * 3.28084

	// Convert wave direction from degrees to cardinal
	wavedegrees, _ := strconv.ParseInt(datafield[11], 0, 64)
	wavecardinal := direction(wavedegrees)

	// Convert water temp from C to F
	watertempC, _ := strconv.ParseFloat(datafield[14], 64)
	watertempF := watertempC*9/5 + 32
	watertempF = RoundPlus(watertempF, 1)

	// Process date/time and convert to PST
	rawtime := strings.Join(datafield[0:5], " ")
	t, err := time.Parse("2006 01 02 15 04", rawtime)
	if err != nil {
		log.Fatal("error processing rawtime:", err)
	}
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatal("error processing location", err)
	}
	t = t.In(loc)

	// Create Observation struct and populate with parsed data
	var o Observation
	o.Date = t
	o.SignificantWaveHeight = waveheightfeet
	o.DominantWavePeriod, err = strconv.Atoi(datafield[9])
	if err != nil {
		log.Fatal("o.AveragePeriod:", err)
	}
	o.AveragePeriod, err = strconv.ParseFloat(datafield[10], 64)
	if err != nil {
		log.Fatal("o.AveragePeriod:", err)
	}
	o.MeanWaveDirection = wavecardinal
	o.WaterTemperature = watertempF

	return o
}

// Given Observation, returns formatted text for tweet
func formatObservation(o Observation) string {
	output := fmt.Sprint(o.Date.Format(time.RFC822), "\nSwell: ", strconv.FormatFloat(float64(o.SignificantWaveHeight), 'f', 1, 64), "ft at ", o.DominantWavePeriod, " sec from ", o.MeanWaveDirection, "\n", "Water: ", o.WaterTemperature, "F")
	return output
}

// Given degrees returns cardinal direction or error message
func direction(deg int64) string {
	switch {
	case deg < 0:
		return "ERROR - DEGREE LESS THAN ZERO"
	case deg <= 11:
		return "N"
	case deg <= 34:
		return "NNE"
	case deg <= 56:
		return "NE"
	case deg <= 79:
		return "ENE"
	case deg <= 101:
		return "E"
	case deg <= 124:
		return "ESE"
	case deg <= 146:
		return "SE"
	case deg <= 169:
		return "SSE"
	case deg <= 191:
		return "S"
	case deg <= 214:
		return "SSW"
	case deg <= 236:
		return "SW"
	case deg <= 259:
		return "WSW"
	case deg <= 281:
		return "W"
	case deg <= 304:
		return "WNW"
	case deg <= 326:
		return "NW"
	case deg <= 349:
		return "NNW"
	case deg <= 360:
		return "N"
	default:
		return "ERROR - DEGREE GREATER THAN 360"
	}
}

// Round input to nearest integer given Float64 and return Float64
func Round(f float64) float64 {
	return math.Floor(f + .5)
}

// RoundPlus truncates a Float64 to a specified number of decimals given Int and Float64, returning Float64
func RoundPlus(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return Round(f*shift) / shift
}
