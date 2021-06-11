package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/tekkamanendless/doiliveinademocracy"
)

func main() {
	port := "8080"
	if value := os.Getenv("PORT"); value != "" {
		port = value
	}
	functionName := "doiliveinademocracy"
	if value := os.Getenv("FUNCTION"); value != "" {
		functionName = value
	}
	customDomain := false
	if value := os.Getenv("CUSTOM_DOMAIN"); value != "" {
		newValue, err := strconv.ParseBool(value)
		if err != nil {
			panic(err)
		}
		customDomain = newValue
	}

	fmt.Printf("Port: %s\n", port)
	fmt.Printf("Function name: %s\n", functionName)
	fmt.Printf("Custom domain: %t\n", customDomain)

	address := ":" + port
	prefix := "/" + functionName

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("REQUEST [IN]\n")
			doiliveinademocracy.CloudFunction(w, r)
			fmt.Printf("REQUEST [OUT]\n")
		})

	serveMux := http.NewServeMux()
	if !customDomain {
		// When we are NOT using a custom domain, then Google quietly trims the function name from the path.
		// This means that when we host the function, we should strip that prefix.
		serveMux.Handle(prefix, http.StripPrefix(prefix, handler))
		serveMux.Handle(prefix+"/", http.StripPrefix(prefix+"/", handler))
	} else {
		// When we ARE using a custom domain (at least through Firebase), the original request's full path
		// is sent to the function.  This menas that we should pass everything to the function as-is.
		serveMux.Handle(prefix, handler)
		serveMux.Handle(prefix+"/", handler)
	}
	serveMux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("INVALID REQUEST\n")
		w.WriteHeader(http.StatusNotFound)
	}))

	server := &http.Server{
		Addr:    address,
		Handler: serveMux,
	}

	fmt.Printf("Listening on: %s\n", address)
	fmt.Printf("Google cloud function: %s\n", functionName)
	fmt.Printf("URL: http://localhost%s%s\n", address, prefix)

	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
