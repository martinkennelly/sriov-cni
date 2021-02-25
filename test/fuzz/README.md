# sriov-fuzzy

- [sriov-fuzzy](#sriov-fuzzy)
  - [Purpose](#purpose)
  - [Build](#build)
  - [Usage](#usage)
  - [Config file](#config-file)
  - [Summary](#summary)

## Purpose

This program will test SRIOV-CNI using input malformed by [Radamsa](https://gitlab.com/akihe/radamsa). Radamsa has to be installed as described in [Radamsa's readme](https://gitlab.com/akihe/radamsa/-/blob/develop/README.md#nutshell).

## Build

```
go build sriov-fuzzy.go
```

## Usage

Usage can be obtained using `./sriov-fuzzy --help`. Please rememeber that it is **mandatory** to specify `device` or `config` option.

```
Usage of ./sriov-fuzzy:
  -cni string
    	Provides path to CNI executable (default "/opt/cni/bin/sriov")
  -config string
    	Config to be used (device value would be ignored if specified)
  -device string
    	Test device PCI address
  -out string
    	Log file path for successful attempts (default "./sriov-fuzzy.log")
  -panicOnly
    	Log only Go panics
  -tests int
    	Number of tests to conduct (default 100000)

```

Example:

```
./sriov-fuzzy --tests 100 --cni /root/sriov --device 0000:02:02.0 --out /var/log/sriov-fuzzy.log
```

Will perform `100` tests of CNI located in `/root/sriov` using device `0000:02:02.0` and will store `all output` in `/var/log/sriov-fuzzy.log`.

```
./sriov-fuzzy --tests 1000 --config config.json
```
Will perform `1000` tests of CNI located in `/opt/cni/bin/sriov` using device config in `config.json` file and will store `all output` in `./sriov-fuzzy.log`.

## Config file

Program can load config file in JSON format. Example config file is provided here:

```json
{
"cniVersion": "0.3.0",
"deviceID":"0000:02:02.1",
"name": "sriov-net-test",
"spoofchk":"off",
"type": "sriov"
}
```

## Summary

When tests are finished program will show test summary:

```
# ./sriov-fuzzy --device 0000:02:02.0 --tests 1000
Performed 1000 tests of which:
889 failed
111 succeeded

Errors by error code:
Code:  100 	Errors:	 691
Code:  1 	Errors:	 198

More details can be found in ./sriov-fuzzy.log
```