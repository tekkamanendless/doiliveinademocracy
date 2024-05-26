package endpoints

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Endpoint represents the endpoints.
type Endpoint struct {
	Prefix string // If this is set, then we'll quietly strip this prefix from the path.  This is helpful for Firebsae function "rewrite" rules.
}

// Handle is an HTTP Cloud function.
func (e *Endpoint) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logrus.WithContext(ctx).Infof("Original path: %s", r.URL.Path)

	// If the path has a trailing slash, then we want to capture that information so that
	// we can restore it when we go to handle the request.
	suffix := ""
	if strings.HasSuffix(r.URL.Path, "/") {
		suffix = "/"
	}

	// Trim the leading and trailing slashes from the path.
	path := strings.Trim(r.URL.Path, "/")
	if prefix := strings.Trim(e.Prefix, "/"); prefix != "" {
		logrus.WithContext(ctx).Infof("Stripping prefix: %s", prefix)
		if path == prefix {
			path = ""
		} else if strings.HasPrefix(path, prefix+"/") {
			path = strings.TrimPrefix(path, prefix+"/")
		}
	}
	logrus.WithContext(ctx).Infof("Path: %s", path)
	logrus.WithContext(ctx).Infof("Suffix: %s", suffix)

	switch path {
	case "":
		handleMainEndpoint(w, r)
		return
	case "_debug":
		handleDebugEndpoint(w, r)
		return
	}

	contents := []byte("Not found")
	w.WriteHeader(http.StatusNotFound)
	w.Write(contents)
}

func handleDebugEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	payload := map[string]interface{}{
		"headers": r.Header,
	}
	contents, err := json.Marshal(payload)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Could not marhsal JSON: %v", err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(contents)
}

// TODO: Do this right.
var once sync.Once
var countryNameByCode map[string]string    // See: http://country.io/names.json
var democracyList []map[string]interface{} // See: https://worldpopulationreview.com/country-rankings/democracy-countries

func handleMainEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Initialize our data if we haven't already.
	once.Do(func() {
		if countryNameByCode == nil {
			logrus.WithContext(ctx).Infof("Loading country data.")
			{
				contents, err := ioutil.ReadFile("data/countries.json")
				if err != nil {
					logrus.WithContext(ctx).Errorf("Could not load country data: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("Could not load country data: %v", err)))
					return
				}
				err = json.Unmarshal(contents, &countryNameByCode)
				if err != nil {
					logrus.WithContext(ctx).Errorf("Could not parse country data: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("Could not parse country data: %v", err)))
					return
				}
			}
		}

		if democracyList == nil {
			logrus.WithContext(ctx).Infof("Loading democracy data.")
			{
				contents, err := ioutil.ReadFile("data/democracies.json")
				if err != nil {
					logrus.WithContext(ctx).Errorf("Could not load country data: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("Could not load country data: %v", err)))
					return
				}
				err = json.Unmarshal(contents, &democracyList)
				if err != nil {
					logrus.WithContext(ctx).Errorf("Could not parse country data: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("Could not parse country data: %v", err)))
					return
				}
			}
		}
	})

	if countryNameByCode == nil {
		logrus.WithContext(ctx).Errorf("There is no country data.")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("There is no country data"))
		return
	}
	if democracyList == nil {
		logrus.WithContext(ctx).Errorf("There is no democracy data.")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("There is no democracy data"))
		return
	}

	var countryCode string
	var countryName string
	var democracyFound bool
	var score float64
	var category string
	answer := "Unknown"
	const minimumScore = 8.01

	countryHeaders := []string{
		"X-Country-Code",
		"X-Appengine-Country",
	}
	for _, header := range countryHeaders {
		value := r.Header.Get(header)
		logrus.WithContext(ctx).Infof("Header: %s: %s", header, value)
	}

	for _, header := range countryHeaders {
		value := r.Header.Get(header)
		if value != "" {
			countryCode = value
			break
		}
	}
	logrus.WithContext(ctx).Infof("Country code: %s", countryCode)

	if countryCode != "" {
		countryName = countryNameByCode[countryCode]
	}
	logrus.WithContext(ctx).Infof("Country name: %s", countryName)

	if countryName != "" {
		for _, democracy := range democracyList {
			if fmt.Sprintf("%v", democracy["country"]) == countryName {
				democracyFound = true

				scoreString := democracy["democracyCountries_score2024"]
				logrus.WithContext(ctx).Infof("Score string: %s", scoreString)

				category = fmt.Sprintf("%v", democracy["democracyCountries_category"])
				logrus.WithContext(ctx).Infof("Category: %s", category)

				var err error
				score, err = strconv.ParseFloat(fmt.Sprintf("%v", scoreString), 64)
				if err != nil {
					logrus.WithContext(ctx).Errorf("Could not parse democracy score: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("Could not parse democracy score: %v", err)))
					return
				}

				break
			}
		}
	}
	logrus.WithContext(ctx).Infof("Score: %f", score)

	if democracyFound {
		if score >= minimumScore {
			answer = "Yes"
		} else {
			answer = "No"
		}
	}

	var contentType string
	var contents string
	if r.URL.Query().Get("mode") == "plain" {
		contentType = "text/plain"
		contents = answer
	} else {
		contentType = "text/html"
		contents = "<!DOCTYPE html>\n"
		contents += "<html>\n"
		contents += "<head>\n"
		contents += "<title>Do You Live In A Democracy?</title>\n"
		contents += "</head>\n"
		contents += "<body>\n"
		contents += "<h1>" + answer + "</h1>\n"
		contents += "<div>\n"
		contents += "Country code: " + countryCode + "<br>\n"
		if countryName == "" {
			contents += "We could not find your country.<br>\n"
		} else {
			contents += "Country name: " + countryName + "<br>\n"
			if !democracyFound {
				contents += "We could not find your democracy score.<br>\n"
			} else {
				contents += "Democracy score: " + fmt.Sprintf("%0.2f", score) + " (" + category + ")<br>\n"
				contents += "<i>Countries scoring " + fmt.Sprintf("%0.2f", minimumScore) + " or higher are considered democracies.  See <a href=\"https://worldpopulationreview.com/country-rankings/democracy-countries\">this link</a> for more details.</i><br>\n"
			}
		}
		contents += "</div>\n"
		contents += "</body>\n"
		contents += "</html>\n"
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(contents))
}
