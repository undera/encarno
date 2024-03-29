# Encarno - The Efficient Load Generator

The name comes from portuguese _[encarno](https://en.wiktionary.org/wiki/encarno) (/əŋˈkar.nu/)_ and means
roughly "[I impersonate](#history)".

## Key Features

- HTTP 1.1 protocol testing, TLS supported, IPv6 supported
- flexible load profiles in ["open" and "closed" workload](https://www.google.com/search?q=open+closed+workload) modes
- accurate load generating up to tens of thousands hits/s
- precise result measurements of nanosecond resolution
- efficient and low overhead (written in Go)
- minimalistic scripting, with regex extracts and asserts


## Usage as Taurus Module

The easiest way to get started is to install [the Python package](https://pypi.org/project/encarno/) using `pip`, which will install also Taurus if needed:

```shell
pip install encarno
```

After that, running any test with `executor: encarno` will automatically download the appropriate version of the Encarno binary. In case you need to point the tool to a custom binary, use this config snippet:
```yaml
modules:
  encarno: 
    path: /my/custom/encarno
```

To run the test, use the usual Taurus command-line with config files. See below for the config examples:
```shell
bzt my-config-with-encarno.yml
```

[Docker image](https://hub.docker.com/r/undera/encarno) is also available for containerized environments: 

```shell
docker run -it -v `pwd`:/conf undera/encarno /conf/config.yml
```

### Closed Workload Mode

Closed workload is the load testing mode when relatively small pool of workers hit the service _as fast as they can_. As service reaches the bottleneck, the response time grows and workers produce less and less hits per second. This kind of workload is typical for service-to-service communications inside cluster. 

In typical tests, the number of workers gradually increases over time to reveal the _capacity limit_ of the service. The result of such test is a _scalability profile_ for the service, also offering the estimation of throughput limits for the [open workload](#open-workload-mode) tests. 

The Taurus config file for closed workload using Encarno:

```yaml
---
execution:
  - executor: encarno
    scenario: simple
    
    concurrency: 50
    ramp-up: 5m
    # steps: 10  # breaks ramp-up into N flat steps

scenarios:
  simple:
    requests: 
      - http://service.net:8080/api/path
```

Note that `hold-for` and `iterations` load profile options are also supported, if you need them. Scenario definition can be [as sophisticated as you need it](#scripting-capabilities).

### Open Workload Mode

Open workload reflects public service scenario, when the number of clients is so big that slowing responses do not lead to decrease in service requests. This is achieved in tests by using large pool of workers that hit service according to _requests schedule_. Usually, that schedule is growing linearly, to reveal the breaking point of the service. Or a steady rate is applied to measure _performance quality characteristics_ for the service, such as response time percentiles.

The main value we configure for open workload tests is the `throughput`, which is the number of requests per second to perform. For the breaking point (aka _stress test_) scenarios we configure it above the [capacity limit](#closed-workload-mode) (~factor x1.5), for quality measurement we aim below the limit (~factor 1/2 or 80%). Usually we also put some limit on possible worker count `concurrency`, due to RAM/CPU being finite for load generator machine.

Stress test config example:

```yaml
---
execution:
  - executor: encarno
    scenario: simple
    
    concurrency: 5000  # it's now the limit, not desired level
    
    throughput: 25000  # hit/s beyond server's capacity
    ramp-up: 5m
    # steps: 10  # breaks ramp-up into N flat steps

scenarios:
  simple:
    requests: 
      - http://service.net:8080/api/path
```

Quality measurement config:

```yaml
---
execution:
  - executor: encarno
    scenario: simple

    concurrency: 5000  # it's now the limit, not desired level

    throughput: 10000  # hit/s below breaking point
    ramp-up: 1m        # should be much shorter than hold-for
    hold-for: 20m      # enough time to accumulate statistics

scenarios:
  simple:
    requests: 
      - http://service.net:8080/api/path
```

### Scripting Capabilities

There are 3 ways to specify inputs for Encarno test: YAML definition via `requests`, URLs list via `requests` and externally generated input file. 

#### Requests Defined in YAML 
The YAML definition looks like [typical Taurus script](https://gettaurus.org/docs/ExecutionSettings/#Scenario) in `requests` section under `scenarios`:

```yaml
scenarios:
  simple:
    default-address: http://i-am-used-by-default:8000
    timeout: 5s

    variables:
      var1: someval

    headers:
      X-My-Global-Header: for all requests

    requests:
      - /assumes-default-address
      - https://full-url-possible/here

      - label: extended detailed request
        url: /path
        method: POST
        headers:
          X-One-More-Header: or many
          Content-Type: application/json
        body: '{"can be": "like this"}'
        # body-file: some.json  # alternative to inline `body`

      - label: regex features, source
        url: /?variable=${var1}
        extract-regexp:
          etag: 'ETag: (".+")'
          date: 'Date: ([^\n]+)'        

      - label: regex features, destination
        url: /path
        method: POST
        headers:
          If-None-Match: '${etag}'
        body: "date was ${date}"
        assert:
          - 'HTTP/1.1 200 OK'
          - not: true
            contains:
              - 'error'
```

Note that `timeout` is only supported on the global level, affecting all the requests equally.

The `variables`, `assert` and `extract-regexp` features work on the full request and response payload text, without breakdown into URI/status/headers/body. Note that variable and regexp usage _will_ make your tests to work a bit slower, due to the processing overhead. Also some more RAM will be used by the load generator.

For HTTP, there is special `:content-length:` variable to be used to obtain correct body length when variables usage alters it dynamically. Taurus module will automatically use that variable when generating POST requests.

#### URLs From Text File

There are the cases when you have a long list of URLs parsed from `access.log` of your server, or dumped from database, or generated by some script. In that situation, you can specify the file containing URLs as value for `requests` option:

```yaml
scenarios:
  simple:
    default-address: http://i-am-used-by-default:8000
    timeout: 5s
    
    headers:
      X-My-Global-Header: for all requests
      
    requests: urls-file.txt
```

The format of the `urls-file.txt` is trivial. It can be either the list of URLs, one per line, or it can also contain request `label` per URL, divided by space:
```text
/just-url/assumes-default-address
http://full-url/goes/here
the_label_before_space /and/then/url
```

All the global settings from scenario still apply to URLs file case, the HTTP method is assumed to be `GET`. Variable evaluation also works for this kind of input.

#### Custom Payload File

Finally, if you need full control over what is sent over network, you can use script file in Encarno's internal [_payload input_ format](#payload-input-format), which ignores most of other scenario options:

```yaml
scenarios:
  simple:
    script: custom_input.enc
    # input-strings: custom_input.str  # for the indexed strings, if needed
```

### TLS Configuration

For the cases, when connecting to server needs special TLS settings like custom cipher suites or TLS versions, please use the following config snippet:

```yaml
modules:
  encarno:
    tls-config:
      insecureskipverify: true  # allow self-signed and invalid certificates, default: false
      minversion: 771  # min version of TLS, default is 1.2
      maxversion: 772  # max version of TLS, default is 1.3
      tlsciphersuites: # list of cipher suite names to use, optional
        - TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
        - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
        - TLS_RSA_WITH_AES_128_CBC_SHA
        - TLS_RSA_WITH_AES_256_GCM_SHA384
```

Possible values for TLS version:
 - TLS 1.3 = `772`
 - TLS 1.2 = `771`
 - TLS 1.1 = `770`
 - TLS 1.0 = `769`

### Debug Trace Log

If you want to see what exactly were sent to the server and what was the response, you may enable the detailed trace log. Encarno would write `encarno_trace.txt` file, containing all meta information, request and response payloads:

```yaml
modules:
  encarno:
    trace-level: 500  # defaults to 1000
```

The trace option logic is "log records which status code is greater or equal X". All network level errors have special code `999`. Be careful with trace level, it can produce large files quickly and run you out of disk space. It is recommended to only enable tracing for certain kind of errors. 

Some level examples:

- `999` or `600` - only the network level errors
- `500` - all 5xx server errors plus network level errors
- `400` - client-side, server-side, and network level
- `0` - dump all the traffic
- `1000` - don't write trace file


### Sidebar Widget

When Taurus displays [console dashboard](https://gettaurus.org/docs/ConsoleReporter/), Encarno provides additional information in its sidebar widget:
- `X wait` - the number of workers that are waiting to get input payload/schedule, if above zero - your load generator is at capacity
- `X busy` - the number of workers actively doing job, synonym to concurrency
- `X sleep` - number of workers waiting for the right time to query (only for open workload), if zero - your load generator is likely at capacity
- `X lag` - the average lag between scheduled time to request and actual time (only for open workload), if above zero - your load generator is likely at capacity 

Please note that widget information may be ahead of aggregate statistics, due to Taurus reporting facility still crunching numbers.

## Standalone Usage

Encarno tool is designed to be used as part of some wrapper (e.g. [Taurus](https://gettaurus.org/)), thus it does not contain much features for result processing and input configuration. If you still want to use the tool on the lower level, this section is for you.

### Building from Source

To build the binary: `go build -o bin/encarno cmd/encarno/main.go`

### Config Format

Here's the full config snippet with some inline comments:
```yaml
input:
    payloadfile: ""      # path to payload input file, mandatory
    iterationlimit: 0    # if above zero, limits number of times the payload file is looped over
    stringsfile: ""      # if specified, contains string index for payload file
    enableregexes: false # enables regex related processing
output:
    ldjsonfile: ""      # optional, path to results file in LDJSON format
    reqrespfile: ""     # optional, path to detailed trace file
    reqrespfilelevel: 0 # trace level for the above option
    binaryfile: ""      # optional path to binary results file, also needs strings file if specified
    stringsfile: ""     # for the binary file, the place to write output string index
    
workers:
    mode: ""            # mandatory workload mode, values are 'open' or 'closed'
    workloadschedule:   # mandatory, the list of linear chunks of workload schedule
        - levelstart: 0 # starting level for chunk
          levelend: 10  # ending level for chunk
          duration: 5s  # duration of chunk
    startingworkers: 0  # optional, number of initial workers to spawn in open workload 
    maxworkers: 0       # the limit of workers to spawn

protocol:
    driver: ""        # mandatory, protocol type to use, defaults to 'http', can also be 'dummy' 
    maxconnections: 0 # limit of connections per host in HTTP
    timeout: 0s       # operation timeout
    tlsconf:          # TLS custom settings
        insecureskipverify: false
        minversion: 0
        maxversion: 0
        tlsciphersuites: []
```

### Payload Input Format

The format is like that because of possible binary payloads. It starts with single-line JSON of metadata, ending with `\n`, then `plen` number of bytes, followed by any number of `\r`, `\n` or `\r\n`.

```text
{"plen": 53, "address": "http://localhost:8070", "label": "/"}
GET / HTTP/1.1
Host: localhost:8070
X-Marker: value


{"plen": 61, "address": "http://localhost:8070", "label": "/gimme404"}
GET /gimme404 HTTP/1.1
Host: localhost:8070
X-Marker: value


```

The metadata may contain optional fields for variable evaluation. Below is formatted JSON of metadata for easier understanding:
```json5
{
  "plen": 0,     // required, payload length
  "label": "",   // item label for grouping in analysis
  "address": "", // address for service under test
  
  "replaces": ["var1", "var2"], // list of variables to evaluate inside payload

  "extracts": {
    "varname": {    // assign the result to this variable name
      "re": ".+",   // apply this regular expression
      "matchNo": 0, // take this match from results, -1 means random
      "groupNo": 0  // take specific capture group from matched regex
    }
  },
  "asserts": [
    {
      "re": ".+",     // regex that must exist in response data
      "invert": false // invert the assertion, it would fail if regex is found
    }
  ]
}
```


The default Taurus configuration would write additional _strings index_ `.istr` file and use `a` and `l` options with string numbers. This is done to minimize the resource footprint. In case you want to see the payload file generated by Taurus without _indexed strings_, use following option:
```yaml
modules:
  encarno:
    index-input-strings: false  
```



### Results Output Formats
Special code 999 is used for network-level errors.

It is possible to switch Encarno from default _binary+strings_ format of output file, into single human-readable LDSON file. It is done via special option:
```yaml
modules:
  encarno:
    output-format: ldjson  # by default, it's "bin"
```

## History

It is written as a replacement for the old [phantom](https://github.com/yandex-load/phantom)+[yandex-tank](https://github.com/yandex/yandex-tank) combination.Those were too "phantom" (and too unmaintained), we're
trying to be "in flesh" analogue to it. The idea was to write a tool as precise as phantom, but using modern programming
language (Go) and address wider spectrum of use-cases.

During implementation, it became apparent that some of phantom's concepts are not as important, namely pre-generated
input file with schedule and payloads. Also, re-implementing HTTP protocol client was considered as an overkill. Maybe
we have lost some speed because of that (we believe not drastically).

It is intentionally not fully-capable _load testing tool_, it is just _load generator_ that assumes the input preparations and result analysis is done by wrapper scripts.

---

## Changelog

### 0.5.3 -- 29 aug 2022
- requests from URLs file have default empty label, to avoid garbaging reports with too many labels
- retry reading string index file


### 0.5.1 -- 25 jul 2022
- regular expression evaluation, extractions and assertions
- make sure custom payload files work fine
- add external payload indexed example

### 0.4 -- 21 jun 2022
- binary output writer&reader, including strings externalization, helper tools to translate into human-readable
- fixes for ipv6, removed internal round-robin cache for DNS for now


### 0.2 and 0.3 -- 13 jun 2022
* improve automated release process: pypi package, docker image

### 0.1 -- 13 jun 2022
* add binary releases on GitHub
* add documentation
* auto-download binary (needs next release)

### 0.0 -- 10 jun 2022
* Simple CLI with one config file
* Open and closed workload support
* HTTP and dummy protocol types
* Input file with metadata in JSON line and full payload
* LDJSON output format
* Log file with health stats
* Taurus module with basic scripting

## Roadmap

- respect `iterations` option from Taurus config, test it, handle "only iterations and no duration is specified"

- when workers decrease (input exhausted or panics), reflect that in counters
- unit tests and coverage
 
- separate file for health status, with per-line flush?
- fully binary input? is it worth it?
- global method and body for urls-from-file case?
- explain log file health KPIs meanings 
- Explain the difference from JMeter and others: How less flexible it is for JMeter, How more flexible it is for Hay and alikes
- document output file formats
- document indexed strings format


### Parking lot

- auto-USL workload
- limit len of auto-label for long GET urls
- udp protocol nib
- Go plugins used for nib

---

<p align="center"><sup><sub><img src="https://raw.githubusercontent.com/lipis/flag-icons/main/flags/4x3/pt.svg" height="8" alt="PT"> Made in Portugal</sub></sup></p>
