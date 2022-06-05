# incarne 
- `phantom` was too "phantom", we're trying to be "in flesh", just working name, we can change it
- can be `scribe` because of `nib`? more competition on GH

## Vision
- replacement for phantom, high-throughput hit-based, for the price of flexible scripting
- binary in, binary out, helper tools to translate into human-readable
- changeable "tip of the spear" `nib` - dummy, http, https, others
- Go plugins used for nib
- included stepper for basic scenarios
- included auto-USL finder driver?
- pluggable scenario component?
- what about scripting? each worker consumes file sequentially and dies after? with some meta-info inside
- no need in pre-calculated stpd - native code would be good at calculating the schedule, though can be one of possible scenarios
- make sure less memory is used => send from disk into socket
- make pluggable into Taurus first
- k8s injector helper tool? Would need integration with Taurus for reporting?

## From @Doctornkz
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

## Entities

- scenario
  - stpd
  - auto-stpd
  - auto-USL
- worker
  - simple
  - scenario-capable (looks at tag-to-regex mapping)
    - how to handle cookie for that? sugar?
    - regex asserts?
- nib
  - dummy
  - http
  - https
  - udp


Test run: `PYTHONPATH=taurus bzt taurus/incarne-module.yml taurus/test.yml -report`

To build the binary: `go build -o bin/incarne cmd/incarne/main.go`

Dummy config:
```yaml
---
execution:
  - executor: incarne
    scenario: simple
    concurrency: 10
    ramp-up: 30s
    hold-for: 1m

scenarios:
  simple:
    protocol: dummy


services:
  - module: shellexec
    prepare:
      - go build -o bin/incarne cmd/incarne/main.go

modules:
  incarne:
    path: bin/incarne

```


## TODO

- graceful shutdown: wait for output to flush and close, interrupt workers, close input
- open workload math
- health stats
- scripting elements in input, whole scripting flow
- URLs from file input type (access.log)
- explicit option of shared input. To allow processing payload file only once.