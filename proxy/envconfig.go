package proxy

import (
	"os"
	"strconv"
	"log"
)
// Env holds all environment variables that can be used to configure the service broker proxy
type Env struct {
	namespace              string
	timeoutSeconds         int
	serviceManagerURL      string
	serviceManagerUser     string
	serviceManagerPassword string
}

func getEnv(name string) string {
	result := os.Getenv(name)
	if len(result) == 0 {
		msg := "Environment variable " + name + " not set."
		log.Fatal(msg)
		panic(msg)
	}
	return result
}

func getEnvInt(name string) int64 {
	envString := getEnv(name)
	result, err := strconv.ParseInt(envString, 10, 64)
	if err != nil {
		msg := "Environment variable " + name + " is not an integer."
		log.Fatal(msg)
		panic(msg)
	}
	return result
}

// EnvConfig creates a new struct Env containing all environment configuration for the service broker proxy
func EnvConfig() Env {
	return Env{
		getEnv("namespace"),
		int(getEnvInt("service_manager_timeout")),
		getEnv("service_manager_url"),
		getEnv("service_manager_user"),
		getEnv("service_manager_password"),
	}
}
