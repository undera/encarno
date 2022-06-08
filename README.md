# encarno 
- the old [phantom](https://github.com/yandex-load/phantom) tool was too "phantom" (and not maintained), we're trying to be "in flesh", https://en.wiktionary.org/wiki/encarno
- 


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


Test run: `PYTHONPATH=taurus bzt taurus/encarno-module.yml taurus/test.yml -report`

To build the binary: `go build -o bin/encarno cmd/encarno/main.go`



## TODO

- http://[::1]:8070/ - should work fine
- binary output writer&reader, including strings externalization
- scripting elements in input, whole scripting flow
- option to inject into k8s
  - inject all the files
  - options to choose NS
  - https://github.com/kubernetes-client/python
  - Download artifacts back https://stackoverflow.com/questions/59703610/copy-file-from-pod-to-host-by-using-kubernetes-python-client

## Parking lot
- explicit option of shared input. To allow processing payload file only once.
  - respect `iterations` option from Taurus config, test it

