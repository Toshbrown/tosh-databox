# Databox start-up scripts and CM go rewrite

This is work-in-progress and incomplete do not use

up-to-date as of 0.4.0

## TODO

- ~~UI proxy is missing (added but working but the path needs fixing)~~
- ~~Some API endpoints are missing~~
- ~~CM auth needs porting over (Almost working)~~ Password generation needs adding
- ~~external IP for https certs needs adding~~
- ~~add a container-manger-core-store, use it to store passwords, installed apps, root cert etc (no more writing to random files!!!) cant do this for then cm root cert as its needed to start the store :-(~~
- ~~Odd design choice to have the CM configure then restart its self (the config would be better done outside then CM)~~ (I remember why i did this now it keeps all the databox setup logic in one place)

- Some hard coded vars registries etc

- finish partial refactor or of lig-go-databox
    - Export service client needs finishing
    - need to handel all content formats currently only JSON is supported

- odd paths and setup due to cramming it all into one repo (you may need to add github.com/toshbrown/tosh-databox to your $GOPATH to get IDEs to play)
    - Move into separate repos

- On app driver restart the IP of the containers needs to be updated see (https://github.com/me-box/core-container-manager/blob/master/src/container-manager.js#L284)
    - This is done but on app restart there is an error re registering with the core network
```
    [policy] Policy.substititue 10.0.2.5 for 10.0.2.6
    2018-06-05 17:09:25 +00:00: INF [dns] Dns_service: banned 10.0.2.6 to resolve driver-os-monitor-core-store
    2018-06-05 17:09:25 +00:00: INF [dns] Dns_service: banned 10.0.2.6 to resolve driver-os-monitor-core-store
    2018-06-05 17:09:25 +00:00: INF [dns] Dns_service: banned 10.0.2.6 to resolve driver-os-monitor-core-store
```
- ~~use the new store where possable (SLA persistence)~~
    - ~~add option to flush the store~~
    - need to update core store to 0.0.7 before it can be used!!

- proxy brakes if http2 upgrade is attempted (curl dose this by default)
- proxy has no support for websockets
- trying to installing an app before any stores causes a hang (simple fix need to return an empty array)
- /run/secrets/DATABOX_ROOT_CA contains the RSA PRIVATE KEY and is passed to all apps and drivers!!!!

- password is hard coded
- need to generate app qr code

- build the databox command in a container so you dont need go installed
- build the databox command for ARM as well as x86

- test and fix locally installed apps for development

- Add filtering to the the new combined log output
- Disable debug output by default (and add a flag to enable it)


# Getting it working

sudo apt install libzmq3-dev - or brew install zmq

go get github.com/toshbrown/tosh-databox

cd ~/go/src/github.com/toshbrown/tosh-databox

Make sure all other version of databox are stopped!

make build && make start

# Stopping it

make stop