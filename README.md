# Encarno - The Efficient Load Generator

The name comes from portuguese _[encarno](https://en.wiktionary.org/wiki/encarno) (/əŋˈkar.nu/)_ and means
roughly "[I impersonate](#history)".

# The Concept

- replacement for phantom, high-throughput hit-based, for the price of flexible scripting
- binary in, binary out, helper tools to translate into human-readable
- changeable "tip of the spear" `nib` - dummy, http, https, others
- Go plugins used for nib
- included stepper for basic scenarios
- included auto-USL finder driver?
- pluggable scenario component?
- what about scripting? each worker consumes file sequentially and dies after? with some meta-info inside
- no need in pre-calculated stpd - native code would be good at calculating the schedule, though can be one of possible
  scenarios
- make pluggable into Taurus first
- k8s injector helper tool? Would need integration with Taurus for reporting?

* 10к rps
* built-in stepper
* post/put/get method
* file-reader
* multi-line ammunition
* labels
* open/closed pattern
* http and net code report
* https/http
* ipv4/ipv6
* answ logs

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

- binary output writer&reader, including strings externalization
- scripting elements in input, whole scripting flow, asserts
-
- http://[::1]:8070/ - should work fine
- respect `iterations` option from Taurus config, test it
-
- unit tests and coverage
-
- auto-release process, including pip
- documentation

### Parking lot

- limit len of auto-label for long GET urls
- udp protocol nib
- explicit option of shared input. To allow processing payload file only once.
- auto-USL workload
- option to inject into k8s
    - inject all the files
    - options to choose NS
    - https://github.com/kubernetes-client/python
    - Download artifacts
      back https://stackoverflow.com/questions/59703610/copy-file-from-pod-to-host-by-using-kubernetes-python-client

