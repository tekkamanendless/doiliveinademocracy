# DoILiveInADemocracy
This is a Google Cloud Function to answer the question, "Do I live in a democracy?"

### Environment Variables
Environment variables are outlined [here](https://cloud.google.com/functions/docs/env-var#nodejs_10_and_subsequent_runtimes).
The following are available:

* `FUNCTION_TARGET`; Reserved: The function to be executed.
* `FUNCTION_SIGNATURE_TYPE`; Reserved: The type of the function: `http` for HTTP functions, and `event` for background functions.
* `K_SERVICE`; Reserved: The name of the function resource.
* `K_REVISION`; Reserved: The version identifier of the function.
* `PORT`; Reserved: The port over which the function is invoked.

Configuration:

* `SLACK_CHANNEL`; this is the slack channel (without the `#`) to send logs to.
* `SLACK_DEBUG`; if this is `true`, then this will turn on debugging for the Slack client (default: `false`).
* `SLACK_LEVEL`; this is the minimum log level to forward to Slack; see `LOG_LEVEL` for options (default: `error`).
* `SLACK_TOKEN`; this is the Slack API token to use.

## Endpoints

### `GET /`
This is the main endpoint.

Query parameters:

* `mode`; this can be `plain` for simple `Yes`/`No` answers; otherwise this will be HTML.

### `GET /_debug`
This returns some debug information about the request.
In particular, this lists the headers and their values.

## Development

### Emulator
Run the Google Cloud function emulator.

```
go run cmd/server/main.go
```

By default, this will be accessible via `http://localhost:8080/example-gcloud-function`.
You may change the port and function name through environment variables.

Environment variables:

* `CUSTOM_DOMAIN`; whether or not to emulate running from a custom domain (default: `false`).
* `FUNCTION`; the name of the Google Cloud Function (default: `example-gcloud-function`).
* `PORT`; the port number to listen on (default: `8000`).

## Google Cloud Functions
As of Go 1.13, the source code (and thus all static assests) is placed in the `./serverless_function_source_code` directory.
For more information, see [the concepts docs](https://cloud.google.com/functions/docs/concepts/exec#file_system).

## Deploy
Push the function to the cloud.

```
gcloud functions deploy doiliveinademocracy --runtime go113 --trigger-http --entry-point CloudFunction --allow-unauthenticated --project YOUR_PROJECT
```

