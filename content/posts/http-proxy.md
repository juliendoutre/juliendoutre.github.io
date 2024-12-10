---
title: "How does a HTTP(s) proxy work?"
description: "An experiment and side project about HTTPS proxying in Go"
date: "2024-12-09"
---

Let's say you have a process that performs many actions involving dependency resolution and fetching, requesting various APIs, sending telemetry, _etc_.
Something like a CI runner compiling a program for instance.
How can you make sure to catch all artifacts it pulls from the outside (namely the Internet) or data it sends out?

I tried using [eBPF](https://ebpf.io/) and [Frida](https://frida.re/) to achieve this but both proved to be difficult to use:
* eBPF can intercept TCP packets in kernel space but TLS connections are negotiated by programs in userspace
* eBPF (or Frida) can attach probes to programs functions, but dependening on their language, they may use different libs to perform HTTP requests. Supporting all of them is too dauting of a task.

A simpler approach would be to have HTTP requests go through a proxy server.
The `HTTP_PROXY` and `HTTPS_PROXY` environment variables are a _de facto_ standard that many tools abide by.
Setting them will make sure that requests are proxied to the URL their values hold instead of being sent directly to the target.

Setting a couple of environment variables is a good enough trade off on my side if it allows an otherwise transparent setup for the user.
My end goal was to build a tool that one could wrap commands with, to monitor all their HTTP requests and get a report of what APIs were reached and what dependencies where pulled.
So let's get started!

```shell
mkdir proxaudit
cd ./proxaudit
go mod init github.com/juliendoutre/proxaudit
git init
```

{{< alert "github" >}}
You can find the project source code on https://github.com/juliendoutre/proxaudit. Check out the commits history to see each iteration!
{{</ alert >}}

First we need to create a HTTP server that will listen on a given port.

```golang
server := &http.Server{
		Addr:              ":" + strconv.FormatUint(*port, 10),
		Handler:           &handler{logger},
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
}
```

Here `logger` is a `*zap.Logger` and port is an `uint64` value provided by the user through a command line flag.
I set up some default timeouts as a good security practice but the values are not critical to the implementation.

`handler` is a custom struct that implements the `http.Handler` interface defined as:
```golang
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```

I wrote it as:
```golang
type handler struct {
	logger *zap.Logger
}

func (h *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.logger.Info("received request", zap.String("method", req.Method), zap.String("url", req.URL.String()))

	httputil.NewSingleHostReverseProxy(req.URL).ServeHTTP(rw, req)
}
```

The `ServeHTTP` method is not doing much.
It logs the request's HTTP verb and the destination URL before deferring the proxy work to a `httputil.ReverseProxy` struct available in Go's standard library.

If I put everything together in a `main.go` file:
```golang
func main() {
	port := flag.Uint64("port", 8000, "port to listen on")
	flag.Parse()

	logger, err := zap.NewProductionConfig().Build()
	if err != nil {
		log.Panic(err)
	}

	server := &http.Server{
		Addr:              ":" + strconv.FormatUint(*port, 10),
		Handler:           &handler{logger},
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go waitForSignal(server, logger)

	logger.Info("Starting HTTP proxy server...", zap.Uint64("port", *port))

	if err := server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			logger.Warn("HTTP proxy server was stopped", zap.Error(err))
		} else {
			logger.Panic("Failed running HTTP proxy server", zap.Error(err))
		}
	}
}
```

and start the server with `go run ./main.go` it should log something like:
```json
{"level":"info","ts":1733854054.4425678,"caller":"proxaudit/main.go:38","msg":"Starting HTTP proxy server...","port":8000}
```
and then hang, waiting for requests.

{{< alert "none" >}}
`waitForSignal` is a util function I wrote to catch signals such as SIGKILL and gracefully terminate the server:

```golang
func waitForSignal(server *http.Server, logger *zap.Logger) {
	stop := make(chan os.Signal, 2)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	if err := server.Shutdown(context.Background()); err != nil {
		logger.Error("Failed shutting down HTTPS proxy server", zap.Error(err))
	}
}
```
{{</ alert >}}

If I open a shell in another tab and run `http_proxy=http://localhost:8000 curl http://google.com` I can see the proxy logging some requests:
```json
{"level":"info","ts":1733854063.6379368,"caller":"proxaudit/main.go:64","msg":"received request","method":"GET","url":"http://google.com/"}
```

We got a plain HTTP request proxy working!

{{< alert "github" >}}
[Code checkpoint](https://github.com/juliendoutre/proxaudit/tree/0cba491ccc77774c5e18d56325deb5128aa08ad4)
{{</ alert >}}
