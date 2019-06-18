# Service Broker Proxy

[![Build Status](https://travis-ci.org/Peripli/service-broker-proxy.svg?branch=master)](https://travis-ci.org/Peripli/service-broker-proxy)
[![Go Report Card](https://goreportcard.com/badge/github.com/Peripli/service-broker-proxy)](https://goreportcard.com/report/github.com/Peripli/service-broker-proxy)
[![Coverage Status](https://coveralls.io/repos/github/Peripli/service-broker-proxy/badge.svg?branch=master)](https://coveralls.io/github/Peripli/service-broker-proxy)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/Peripli/service-broker-proxy/blob/master/LICENSE)

Framework for writing Service Manager broker proxies.

## Purpose

Contains code to write proxy agents for the Service Manager.
It provides logic for service broker registration and access reconciliation between the Service Manager and the platform that the proxy represents as well as logic for proxying OSB API calls. It's first consumers are `github.com/Peripli/service-broker-proxy-k8s` and `github.com/Peripli/service-broker-proxy-cf`.
