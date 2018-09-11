// Package main of aircam-snapshot provides a tool for maintaining an
// authenticated session with a Ubiquiti AirCam, allowing unauthenticated image
// retrieval.
package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

// Type config represents the configuration for the application, with the names
// of the variables representing their corresponding environment variables.
type config struct {
	URL       string
	Username  string
	Password  string
	IgnoreSSL bool
	Port      int
}

// Package level configuration and http client
var (
	conf   config
	client http.Client
)

func init() {
	// Parse the URL of the AirCam, exiting if undefined
	if URL, err := os.LookupEnv("SNAPSHOT_URL"); err {
		conf.URL = URL
	} else {
		log.Fatal("SNAPSHOT_URL not defined")
	}

	// Parse the username to login to the AirCam with, exiting if undefined
	if username, err := os.LookupEnv("SNAPSHOT_USERNAME"); err {
		conf.Username = username
	} else {
		log.Fatal("SNAPSHOT_USERNAME not defined")
	}

	// Parse the password to login to the AirCam with, exiting if undefined
	if password, err := os.LookupEnv("SNAPSHOT_PASSWORD"); err {
		conf.Password = password
	} else {
		log.Fatal("SNAPSHOT_PASSWORD not defined")
	}

	// Parse the ignore SSL variable, defaulting to no if undefined
	if ignoreSSL, err := os.LookupEnv("SNAPSHOT_IGNORE_SSL"); err {
		switch ignoreSSL {
		case "true":
			conf.IgnoreSSL = true
		case "false":
			conf.IgnoreSSL = false
		default:
			conf.IgnoreSSL = true
		}
	} else {
		conf.IgnoreSSL = true
	}

	// Parse the port variable, defaulting to 8000 if undefined
	if port, err := os.LookupEnv("SNAPSHOT_PORT"); err {
		var parseErr error
		conf.Port, parseErr = strconv.Atoi(port)

		if parseErr != nil {
			log.Fatal("Invalid value for SNAPSHOT_PORT")
		}
	} else {
		conf.Port = 8000
	}

	// Set the ignore SSL setting in the HTTP client
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: conf.IgnoreSSL,
	}
}

func main() {
	// Login to the camera
	sessionCookie, err := login()
	if err != nil {
		log.Fatalf("Login failed: %s", err)
	}

	// Create handler function for retrieving images
	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Print("Getting image")

		// Set the header to indicate image content and retrieve image from AirCam
		w.Header().Set("Content-Type", "image/jpeg")
		getImage(w, sessionCookie)
	}

	// Associate handler
	http.HandleFunc("/snapshot.cgi", handler)

	// Start the HTTP server
	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%d", conf.Port), nil))
}

// getImage retrieves an image from a provided url using a session cookie.
// It returns a byte slice with the image contents.
func getImage(out io.Writer, sessionCookie *http.Cookie) {
	// Byte slice that will eventually hold the image contents
	var image []byte

	// Create an HTTP request based on the provided URL endpoint, returning an
	// error if the request cannot be created.
	request, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/snapshot.cgi", conf.URL), nil)
	if err != nil {
		log.Printf("Image - Error creating request: %s", err)
	}

	// Add the session cookie to the request
	request.AddCookie(sessionCookie)

	// Make the HTTP request with the shared http Client, returning an error if
	// the request fails or times out.
	response, err := client.Do(request)
	if err != nil {
		log.Printf("Image - Error creating response: %s", err)
	}

	// Check if the status code is OK (200) and return an error if it is not.
	if response.StatusCode != http.StatusOK {
		log.Printf("Image - Non-200 status code received: %d", response.StatusCode)
	}

	// Parse the response body into a byte slice, returning an error if unable to
	// parse.
	image, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Image - Error reading response body, %s", err)
	}

	// Return the byte slice.
	out.Write(image)
}

// login performs the login process for an AirCam.
// It returns a session cookie, and any errors encountered during login.
func login() (*http.Cookie, error) {
	log.Printf("Login - Logging in with username \"%s\" and password \"%s\"",
		conf.Username, conf.Password)

	// Make an initial request to the root of the webserver.
	// This is the only URL which provides a session cookie.
	initialURL := fmt.Sprintf("%s/", conf.URL)
	log.Printf("Login - Making initial request to retrieve session cookie: %s",
		initialURL)
	initialRequest, err := http.NewRequest("GET", initialURL, nil)
	initialResponse, err := client.Do(initialRequest)

	if err != nil {
		log.Printf("Login - Error making initial request: %s", err)
		return nil, err
	}

	// Locate the session cookie in the response, erroring if not found.
	log.Printf("Login - Finding session cookie")
	var sessionCookie *http.Cookie
	sessionFound := false
	for _, cookie := range initialResponse.Cookies() {
		if cookie.Name == "AIROS_SESSIONID" {
			log.Printf("Login - Found session cookie: %s", cookie.Value)
			sessionCookie = cookie
			sessionFound = true
		}
	}

	if !sessionFound {
		log.Printf("Login - Could not find session cookie")
		return nil, errors.New("Login - Could not find session cookie")
	}

	// Create a multipart form body
	log.Print("Login - Constructing multipart form data")

	// Byte buffer to hold the body
	bodyBuffer := &bytes.Buffer{}

	// Multipart writer
	bodyWriter := multipart.NewWriter(bodyBuffer)

	// Construct map containing the form fields and their values
	formValues := map[string]string{
		"uri":      "/snapshot.cgi",
		"Submit":   "Login",
		"username": conf.Username,
		"password": conf.Password,
	}

	// Write each field and value to the multipart writer
	for field, value := range formValues {
		err = bodyWriter.WriteField(field, value)

		if err != nil {
			log.Printf("Login - Error encoding field %s with value %s: %s", field,
				value, err)
			return nil, err
		}
	}

	bodyWriter.Close()

	// Make the request to the login endpoint on the AirCam.
	loginURL := fmt.Sprintf("%s/login.cgi", conf.URL)
	log.Printf("Login - Creating login request: %s", loginURL)

	// Create a new POST request to the login endpoint with the multipart buffer
	request, err := http.NewRequest("POST", loginURL, bodyBuffer)

	// Add the session cookie retrieved earlier
	request.AddCookie(sessionCookie)

	// Dynamically set the Content-Type header to indicate the form boundary
	request.Header.Set("Content-Type", bodyWriter.FormDataContentType())

	if err != nil {
		log.Printf("Login - Error creating login request: %s", err)
		return nil, err
	}

	// Make the login request
	log.Print("Login - Making login request")
	response, err := client.Do(request)

	// Check if there was an error making the request or if the server did not
	// respond with 200
	if err != nil {
		log.Printf("Login - Error making login request: %s", err)
		return nil, err
	} else if response.StatusCode != http.StatusOK {
		log.Printf("Login - Error making login request: HTTP %d",
			response.StatusCode)
		return nil, fmt.Errorf("Login - Error making login request: HTTP %d",
			response.StatusCode)
	}

	// Return the session cookie and no error
	return sessionCookie, nil
}
