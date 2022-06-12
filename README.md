# Encarno - The Efficient Load Generator

The name comes from portuguese _[encarno](https://en.wiktionary.org/wiki/encarno) (/əŋˈkar.nu/)_ and means
roughly "[I impersonate](#history)".

## Key Features

- http and https protocol testing
- flexible load profiles in ["open" and "closed" workload](https://www.google.com/search?q=open+closed+workload) modes
- accurate load generating
- precise result measurements of nanosecond resolution
- efficient and low overhead (written in Go)
- minimalistic scripting, via regex extracts and asserts (coming soon)


## Usage as Taurus Module
Test run: `PYTHONPATH=taurus bzt taurus/encarno/encarno-module.yml taurus/test.yml`

### Closed Workload

### Open Workload

### Scripting Capabilities

requests, urls input, own payload file

## Standalone Usage

Kinda meaningless without the wrapper

### Building from Source

To build the binary: `go build -o bin/encarno cmd/encarno/main.go`

### Config Format

### Payload Input Format
The format is like that because of binary payloads
```text
{"PayloadLen": 57, "Address": "http://localhost:8070", "Label": "/"}
GET / HTTP/1.1
Host: localhost:8070
X-Marker: value


{"PayloadLen": 65, "Address": "http://localhost:8070", "Label": "/gimme404"}
GET /gimme404 HTTP/1.1
Host: localhost:8070
X-Marker: value


```

### Results Output Format
special code 999 for errors
### Debug trace log
### Log file health meanings
## Comparison to Similar Tools

Explain the difference from JMeter and others
How less flexible it is for JMeter
How more flexible it is for Hay and alikes

## History

It is written as a replacement for the old [phantom](https://github.com/yandex-load/phantom)
+[yandex-tank](https://github.com/yandex/yandex-tank) combination.Those were too "phantom" (and too unmaintained), we're
trying to be "in flesh" analogue to it. The idea was to write a tool as precise as phantom, but using modern programming
language (Go) and address wider spectrum of use-cases.

During implementation, it became apparent that some of phantom's concepts are not as important, namely pre-generated
input file with schedule and payloads. Also, re-implementing HTTP protocol client was considered as an overkill. Maybe
we have lost some speed because of that (we believe not drastically).

## Changelog

### 0.0 -- 10 jun 2022

* Simple CLI with one config file
* Open and closed workload support
* HTTP and dummy protocol types
* Input file with metadata in JSON line and full payload
* LDJSON output format
* Log file with health stats
* Taurus module with basic scripting

## Roadmap

- binary output writer&reader, including strings externalization, helper tools to translate into human-readable
- scripting elements in input, whole scripting flow, asserts
- when workers decrease (input exhausted or panics), reflect that in counters
- 
- http://[::1]:8070/ - should work fine
- respect `iterations` option from Taurus config, test it
- 
- unit tests and coverage
- 
- auto-release process, including pip
- documentation
- separate file for health status, with per-line flush?

### Parking lot

- limit len of auto-label for long GET urls
- udp protocol nib
- auto-USL workload
- option to inject into k8s
    - inject all the files
    - options to choose NS
    - https://github.com/kubernetes-client/python
    - Download artifacts
      back https://stackoverflow.com/questions/59703610/copy-file-from-pod-to-host-by-using-kubernetes-python-client
- Go plugins used for nib
