#!/usr/bin/env bash

htmlreport="$PWD/test-coverage-report.out.html"
coveragefile="$(mktemp /tmp/coverage.XXXXXXXXX)"

go test -coverprofile="$coveragefile"

echo -e "\nPrinting coverage information:"
go tool cover -func="$coveragefile"

go tool cover -html="$coveragefile" -o "$htmlreport"
echo -e "\nGenerated file $htmlreport with the test coverage report"
