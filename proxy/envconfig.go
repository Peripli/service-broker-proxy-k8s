package proxy

import (
	"strconv"
	"log"
	"io/ioutil"
)
// Env holds all environment variables that can be used to configure the service broker proxy
type Env struct {
	namespace              string
	timeoutSeconds         int
	serviceManagerURL      string
	serviceManagerUser     string
	serviceManagerPassword string
}

func getConfiguration(name string) string {
	result, err := ioutil.ReadFile("/etc/configuration/" + name)
	if err != nil {
		msg := "Configuration " + name + " cannot be read from the volume."
		log.Fatal(msg)
		panic(msg)
	}
	return string(result)
}

func getConfigurationInt(name string) int64 {
	envString := getConfiguration(name)
	result, err := strconv.ParseInt(envString, 10, 64)
	if err != nil {
		msg := "Configuration " + name + " cannot be converted to an integer."
		log.Fatal(msg)
		panic(msg)
	}
	return result
}

func getSecret(name string) string {
	secret, err := ioutil.ReadFile("/etc/service-manager-secrets/" + name)
	if err != nil {
		msg := "Secret " + name + " cannot be read from the volume."
		log.Fatal(msg)
		panic(msg)
	}
	return string(secret)
}

// EnvConfig creates a new struct Env containing all environment configuration for the service broker proxy
func EnvConfig() Env {
	return Env{
		getConfiguration("namespace"),
		int(getConfigurationInt("service_manager_timeout")),
		getSecret("url"),
		getSecret("user"),
		getSecret("password"),
	}
}
