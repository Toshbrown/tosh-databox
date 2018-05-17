# Databox start-up scripts and CM go rewrite

This is work-in-progress and incomplete do not use

up-to-date as of 0.3.2 - any new features/bug fixes will need to be ported

## TODO

UI proxy is missing
Some API endpoints are missing
Some hard coded vars registries etc

Odd design choice to have the CM configure then restart its self (the config would done outside then CM)

# Getting it working

./databox-test

# Stopping it

go run main.go --cmd stop