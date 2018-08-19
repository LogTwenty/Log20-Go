# Log20 - Go

## Approaches

### ast.Walk

### syscall.Ptrace

[Debuggers from Scratch - Liz Rice](https://www.youtube.com/watch?v=TBrv17QyUE0)

https://syslog.ravelin.com/go-function-calls-redux-609fdd1c90fd

https://github.com/derekparker/delve/blob/master/Documentation/usage/dlv_attach.md
https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/derekparker/delve%24%40master+AttachPid#4

## pprof

https://github.com/golang/go/wiki/Performance
https://golang.org/pkg/net/http/pprof/
https://golang.org/pkg/runtime/pprof/
https://github.com/google/pprof/blob/master/doc/README.md
https://github.com/google/gops
https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/
https://www.farsightsecurity.com/2016/10/28/cmikk-go-remote-profiling/

### tracing

https://golang.org/pkg/runtime/trace/

### eBPF

### Log Analysis

## Known Logging Libraries

- stdlib logging
- glog
- log15
- logrus
- zap

## Links

https://golang.org/doc/diagnostics.html
http://www.brendangregg.com/blog/2017-01-31/golang-bcc-bpf-function-tracing.html
https://hackernoon.com/strace-in-60-lines-of-go-b4b76e3ecd64