---
title: "Writing a HTTPS proxy in Golang"
description: "An experiment and side project about HTTPS proxying in Go"
date: "2024-12-09"
---

Let's say you have a process that performs many actions involving dependency resolution and fetching, requesting various APIs, sending telemetry, _etc_.
Something like a CI runner compiling a program for instance.
How can you make sure to catch all artifacts it pulls from the outside (namely the Internet) or data it sends out?

I tried using [eBPF](https://ebpf.io/) and [Frida](https://frida.re/) to achieve this but both proved to be difficult to use:
* eBPF can intercept TCP packets in kernel space but TLS connections are negotiated by programs in user space
* eBPF (or Frida) can attach probes to functions in user space, but depending on the languages at stake, different libs may be used to perform HTTP requests. Supporting all of them is too dauting of a task.

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

Nowadays though, most people use HTTPS. Running `http_proxy=http://localhost:8000 curl https://google.com` does not result in any log on the proxy side. We could have expected this, we changed of protocol. But what happens if I run `HTTPS_PROXY=http://localhost:8000 curl https://google.com`?

The server logs an error: `http: proxy error: unsupported protocol scheme ""` and curl returns an error as well: `curl: (56) CONNECT tunnel failed, response 502`.

Why is this? I understood the issue while reading https://eli.thegreenplace.net/2022/go-and-proxy-servers-part-2-https-proxies/.

A HTTPS connection can go through proxies but will only send a CONNECT request to them, indicating which host it targets. It's expecting the underlying TLS connection to be tunneled as is.

In order to fix my code, I extended my `ServeHTTP` function like this:
```golang
func (h *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.logger.Info("received request", zap.String("method", req.Method), zap.String("url", req.URL.String()))

	if req.Method == http.MethodConnect {
		h.handleConnect(rw, req)
	} else {
		httputil.NewSingleHostReverseProxy(req.URL).ServeHTTP(rw, req)
	}
}

func (h *handler) handleConnect(rw http.ResponseWriter, req *http.Request) {
	hijacker, ok := rw.(http.Hijacker)
	if !ok {
		http.Error(rw, "Hijacking not supported", http.StatusInternalServerError)

		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusServiceUnavailable)

		return
	}
	defer clientConn.Close()

	serverConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusServiceUnavailable)

		return
	}
	defer serverConn.Close()

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go io.Copy(serverConn, clientConn)
	io.Copy(clientConn, serverConn)
}
```

The `handleConnect` menthod really just returns a success status to the client and then recover the underlying TCP connection (thanks to the `http.Hijacker` interface casting) to simply forward all the traffic to the destination host.

Now I can run `HTTPS_PROXY=http://localhost:8000 curl https://google.com` without errors!

{{< alert "github" >}}
[Code checkpoint](https://github.com/juliendoutre/proxaudit/tree/be4704f55d7400d5340b2dd67ccef01e3e88a6ff)
{{</ alert >}}

But... it's simply logging CONNECT requests, not the underlying HTTPS requests. As the proxy blindly forward the TLS connection to the host, it does not have access to the raw content. And it's why TLS is used for, right? Preventing eavesdroppers from intercepting cleartext traffic. Fortunately, there's a way to overcome this though by using Man-In-The-Middle (MITM) certificate crafting.

I read [some documentation](https://docs.mitmproxy.org/stable/concepts-howmitmproxyworks/#explicit-https) from the [mitmproxy](https://mitmproxy.org/) Python project to understand the technique at play. The idea is to craft a fake certificate using a Certificate Authority (CA) under our control and that is trusted by the client to terminate the connection in the proxy. Then open a second connection to the server to forward requests after they've been handled by the proxy. This requires to fetch the destination's certificate to craft a matching certificate.

As my end goal is to make an open source observability tool, I can ask users to trust a certificate that would intercept their traffic. I first ran:
```shell
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt
openssl req -newkey rsa:4096 -nodes -keyout server.key -subj "/CN=localhost" -out server.csr
openssl x509 -req -extfile <(printf "subjectAltName=DNS:localhost") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt
```
to generate a CA and a certificate for my proxy before a colleague told me about https://github.com/FiloSottile/mkcert:
```shell
brew install mkcert
mkcert -install
mkcert localhost
```

I used the excellent https://github.com/elazarl/goproxy Go module to perform the MITM interception for me:
```golang
func main() {
	// [...]

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(goproxy.ReqConditionFunc(
		func(req *http.Request, _ *goproxy.ProxyCtx) bool {
			return req.Method != http.MethodConnect
		},
	)).DoFunc(
		func(req *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			logger.Info("received a request", zap.String("method", req.Method), zap.String("url", req.URL.String()))

			return req, nil
		},
	)

	// [...]
}
```

I configured the proxy to mitm connect requests and log all other methods. I replaced my custom handler with it:
```golang
server := &http.Server{
	Addr:              ":" + strconv.FormatUint(*port, 10),
	Handler:           proxy, // this line changed
	ReadTimeout:       10 * time.Second,
	ReadHeaderTimeout: 5 * time.Second,
	IdleTimeout:       120 * time.Second,
}
```

and loaded the CA I created with mkcert before starting it:
```golang
func main() {
	mkcertDir := path.Join(os.Getenv("HOME"), "Library", "Application Support", "mkcert")

	port := flag.Uint64("port", 8000, "port to listen on")
	caCertPath := flag.String("ca-cert", path.Join(mkcertDir, "rootCA.pem"), "path to a CA certificate")
	caKeyPath := flag.String("ca-key", path.Join(mkcertDir, "rootCA-key.pem"), "path to a CA private key")
	flag.Parse()

	logger, err := zap.NewProductionConfig().Build()
	if err != nil {
		log.Panic(err)
	}

	cert, err := tls.LoadX509KeyPair(*caCertPath, *caKeyPath)
	if err != nil {
		logger.Panic("Failed loading CA certificate", zap.Error(err))
	}

	goproxy.GoproxyCa = cert

	// [...]
}
```

and now HTTPS interception works like a charm!

```json
{"level":"info","ts":1733868127.719048,"caller":"proxaudit/main.go:66","msg":"Starting HTTP proxy server...","port":8000}
{"level":"info","ts":1733868147.167201,"caller":"proxaudit/main.go:50","msg":"received a request","method":"GET","url":"http://google.com/"}
{"level":"info","ts":1733868150.407752,"caller":"proxaudit/main.go:50","msg":"received a request","method":"GET","url":"http://google.com/"}
{"level":"info","ts":1733868150.427733,"caller":"proxaudit/main.go:50","msg":"received a request","method":"GET","url":"http://www.google.com/"}
{"level":"info","ts":1733868156.662025,"caller":"proxaudit/main.go:50","msg":"received a request","method":"GET","url":"https://google.com:443/"}
{"level":"info","ts":1733868160.218765,"caller":"proxaudit/main.go:50","msg":"received a request","method":"GET","url":"https://google.com:443/"}
{"level":"info","ts":1733868160.5086799,"caller":"proxaudit/main.go:50","msg":"received a request","method":"GET","url":"https://www.google.com:443/"}
^C{"level":"warn","ts":1733868173.727085,"caller":"proxaudit/main.go:70","msg":"HTTP proxy server was stopped","error":"http: Server closed"}
```

{{< alert "github" >}}
[Code checkpoint](https://github.com/juliendoutre/proxaudit/tree/969be3daac2d8ec5c1a08549ec4133b456885869)
{{</ alert >}}

Some tools may require more than setting `HTTPS_PROXY` to work. For instance:
```shell
HTTPS_PROXY=https://localhost:8000 npm i dd-trace
```
alone does not work. One need to run:
```shell
NODE_EXTRA_CA_CERTS="$(mkcert -CAROOT)/rootCA.pem" HTTPS_PROXY=https://localhost:8000 npm i dd-trace
```
to allow our local CA to emit trusted certificates.

{{< alert "none" >}}
Note that for old versions of NPM, there was an issue documented at https://github.com/dependabot/dependabot-core/issues/10623 when using goproxy.
{{</ alert >}}

## Introducing proxaudit!

The final stage for me was to use all this logic in a tool that can wrap any command and observe its HTTP(s) requests thanks to a proxy. Check out https://github.com/juliendoutre/proxaudit to see the project's latest version!

```shell
brew tap juliendoutre/proxaudit https://github.com/juliendoutre/proxaudit
brew install proxaudit
mkcert -install
proxaudit -- curl http://google.com
proxaudit -- curl https://google.com
proxaudit # Read from stdin
proxaudit -output logs.jsonl -- curl https://google.com # Write logs to file
```

I took inspiration from  https://github.com/99designs/aws-vault for the design (especially the way they forward signals to subprocesses).

Let me know if anything is missing or if you found a bug by opening [an issue](https://github.com/juliendoutre/proxaudit/issues) :pray:

See you next time :wave:
