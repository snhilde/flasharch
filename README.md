![Maintenance Badge](https://img.shields.io/badge/Maintained-yes-success)
![Class Badge](https://img.shields.io/badge/Status-experimental-yellow)
![Version Badge](https://img.shields.io/badge/Version-1.0-informational)
[![GoReportCard example](https://goreportcard.com/badge/github.com/snhilde/flasharch)](https://goreportcard.com/report/github.com/snhilde/flasharch)

# flasharch
Convenient script for downloading the latest Arch Linux ISO and creating a bootable flash drive

## Installation
Get the source:
```
go get github.com/snhilde/flasharch
```

Install the program into your GOBIN:
```
go install ${GOPATH}/src/github.com/snhilde/flasharch
```
Now, if your GOBIN is part of your PATH, you can run `flasharch` from the command line.

## Usage
Download the latest ISO image for Arch Linux and create a bootable USB drive:
```
flasharch /full/path/to/usb
```
Change `/full/path/to/usb` to the device file of your USB (e.g. `/dev/sdc`). Device files can be discovered with `lsblk`.

## Configuration
The only setting you might want to configure is the mirror holding the ISO file. A full list of mirrors is [here](https://www.archlinux.org/download/), under "HTTP Direct Downloads". Choose one you like, and set is as `var mirror` in [main.go](main.go), right beneath the import statements. Please note that the path in the URL should end in `/iso/latest/` to get the current release. Optionally choose a different directory to flash a previous release.
