# Databox start-up scripts and CM go rewrite

This is work-in-progress and incomplete do not use

up-to-date as of 0.4.0

## TODO

- ~~UI proxy is missing (added but working but the path needs fixing)~~
- ~~Some API endpoints are missing~~
- Some hard coded vars registries etc
- ~~CM auth needs porting over (Almost working)~~ Password generation needs adding
- external IP for https certs needs adding

- Odd design choice to have the CM configure then restart its self (the config would be better done outside then CM)

- finish partial refractor or of lig-go-databox
- odd paths and setup due to cramming it all into one repo (you may need to add github.com/toshbrown/tosh-databox to your $GOPATH to get IDEs to play)
- add a container-manger-core-store, use it to store passwords, installed apps, root cert etc (no more writing to random files!!!)
- proxy brakes if http2 upgrade is attempted (curl dose this by default)
- proxy has no support for websockets
- On app driver restart the IP of the containers needs to be updated see (https://github.com/me-box/core-container-manager/blob/master/src/container-manager.js#L284)
- trying to installing an app befor any stores causes a hang
- /run/secrets/DATABOX_ROOT_CA contains the RSA PRIVATE KEY and is passed to all apps and drivers!!!!


# Getting it working

sudo apt install libzmq3-dev - or brew install zmq

go get github.com/toshbrown/tosh-databox

cd ~/go/src/github.com/toshbrown/tosh-databox

Make sure all other version of databox are stopped!

make build && make build-cm && make start

# Stopping it

make stop