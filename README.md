# Databox start-up scripts and CM go rewrite

This is work-in-progress and incomplete do not use

up-to-date as of 0.3.2 - any new features/bug fixes will need to be ported

## TODO

- UI proxy is missing
- Some API endpoints are missing
- Some hard coded vars registries etc

- Odd design choice to have the CM configure then restart its self (the config would be better done outside then CM)

- finish partial refractor or of lig-go-databox
- odd paths and setup due to cramming it all into one repo (you may need to add github.com/toshbrown/tosh-databox to your $GOPATH to get IDEs to play)

# Getting it working

Make sure all other version of databox are stopped!

./databox-test

# Stopping it

go run main.go --cmd stop