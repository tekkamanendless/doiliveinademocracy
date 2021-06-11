package doiliveinademocracy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/tekkamanendless/doiliveinademocracy/internal/endpoints"
	"github.com/tekkamanendless/doiliveinademocracy/internal/slackhook"
	"github.com/tekkamanendless/gcfstructuredlogformatter"
)

// GoogleCloudFunctionSourceDirectory is where Google Cloud will put the source code that was uploaded.
//
// Note that for Go 1.11, the source code was in the current working directory.
// However, for Go 1.13, they moved it to this directory.
//
// See: https://cloud.google.com/functions/docs/concepts/exec#file_system
const GoogleCloudFunctionSourceDirectory = "serverless_function_source_code"

// once is an object that will only execute its function one time.
//
// Because we want to log during our initialization, we need to handle this in a non-standard
// function and keep track of our initialization status.
var once sync.Once

// Initialize initializes the application.
//
// Primarily, this changes the current working directory.
func Initialize(ctx context.Context) {
	logrus.WithContext(ctx).Infof("Initializing the application.")

	// Handle Slack early on, since the results from this section will dictate how
	// logs are treated.
	{
		slackChannel := os.Getenv("SLACK_CHANNEL")
		slackLevel := os.Getenv("SLACK_LEVEL")
		if slackLevel == "" {
			slackLevel = "error"
		}
		slackToken := os.Getenv("SLACK_TOKEN")
		logrus.WithContext(ctx).Infof("Slack channel: %s", slackChannel)
		logrus.WithContext(ctx).Infof("Slack level: %s", slackLevel)
		if slackToken == "" {
			logrus.WithContext(ctx).Infof("Slack token: n/a")
		} else {
			logrus.WithContext(ctx).Infof("Slack token: ********")
		}
		debugSlack := false
		if value := os.Getenv("SLACK_DEBUG"); value != "" {
			var err error
			debugSlack, err = strconv.ParseBool(value)
			if err != nil {
				logrus.WithContext(ctx).Errorf("Could not parse value %q: %v", value, err)
				os.Exit(1)
			}
		}
		logrus.WithContext(ctx).Infof("Debug slack: %t", debugSlack)
		if slackToken != "" && slackChannel != "" {
			// Parse the slack log level.
			minimumSlackLogLevel, err := logrus.ParseLevel(slackLevel)
			if err != nil {
				logrus.WithContext(ctx).Warnf("Unknown log level: %q", slackLevel)
				minimumSlackLogLevel = logrus.ErrorLevel // Default to the error level.
			}

			slackClient := slack.New(slackToken, slack.OptionDebug(debugSlack))
			hook := slackhook.New(slackClient, slackChannel, minimumSlackLogLevel)
			logrus.AddHook(hook)

			logrus.WithContext(ctx).Infof("Slack hook has been registered.")

			/* Re-enable these to verify the Slack hook is working appropriately.
			logrus.WithContext(ctx).Tracef("Trace")
			logrus.WithContext(ctx).Debugf("Debug")
			logrus.WithContext(ctx).Infof("Info")
			logrus.WithContext(ctx).Warnf("Warn")
			logrus.WithContext(ctx).Errorf("Error")
			logrus.WithContext(ctx).Fatalf("Fatal")
			logrus.WithContext(ctx).Panicf("Panic")
			os.Exit(0)
			//*/
		}
	}

	path, err := os.Getwd()
	if err != nil {
		logrus.WithContext(ctx).Warnf("Could not find the current working directory: %v", err)
	}
	logrus.WithContext(ctx).Infof("Current working directory: %s", path)

	logrus.WithContext(ctx).Infof("Looking for top-level source directory: %s", GoogleCloudFunctionSourceDirectory)
	fileInfo, err := os.Stat(GoogleCloudFunctionSourceDirectory)
	if err == nil && fileInfo.IsDir() {
		logrus.WithContext(ctx).Infof("Found top-level source directory: %s", GoogleCloudFunctionSourceDirectory)
		err = os.Chdir(GoogleCloudFunctionSourceDirectory)
		if err != nil {
			logrus.WithContext(ctx).Warnf("Could not change to directory %q: %v", GoogleCloudFunctionSourceDirectory, err)
		}
	}

	logrus.WithContext(ctx).Infof("Initialization complete.")
}

// CloudFunction is an HTTP Cloud Function with a request parameter.
func CloudFunction(w http.ResponseWriter, r *http.Request) {
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		traceHeader := r.Header.Get("X-Cloud-Trace-Context")
		traceParts := strings.Split(traceHeader, "/")
		if len(traceParts) > 0 && len(traceParts[0]) > 0 {
			trace := fmt.Sprintf("projects/%s/traces/%s", projectID, traceParts[0])
			r = r.WithContext(context.WithValue(r.Context(), gcfstructuredlogformatter.ContextKeyTrace, trace))
			logrus.WithContext(r.Context()).Debugf("Trace: %s", trace)
		}

		formatter := gcfstructuredlogformatter.New()

		logrus.SetFormatter(formatter)
	}

	ctx := r.Context()

	// Initialize our application if we haven't already.
	once.Do(func() { Initialize(ctx) })

	path, err := os.Getwd()
	if err != nil {
		logrus.WithContext(ctx).Warnf("Could not find the current working directory: %v", err)
	}
	logrus.WithContext(ctx).Infof("Current working directory: %s", path)

	e := endpoints.Endpoint{
		Prefix: "doiliveinademocracy", // This will be the path that we host the function under (in Firebase).
	}
	e.Handle(w, r)
}
