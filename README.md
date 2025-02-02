# API key limiter
API Key Limiter is a proxy intended to help manage and rate-limit API key usage. Users can use the [configuration management tool](https://github.com/IgorPidik/api-key-limiter-conf) to create configurations, where they define headers and values to be replaced in the incoming requests.
These requests are then forwarded to the original destination.

## How does it work
API Key Limiter is a Man-In-The-Middle proxy, this configuration is necessary, otherwise the proxy would be unable to inspect and alter the request headers.
However, this means that all the headers and the body of both the request and the response are exposed to the proxy. This presents security risks and should be considered before usage.

## How to run
### 1. Setup the configuration management tool
First, you need to setup the [configuration management tool](https://github.com/IgorPidik/api-key-limiter-conf), which will allow you to create configurations. These define headers and values to be replaced in the requests. Afterwards, we can setup the proxy.

### 2. Create configurations
Use the configuration management tool to create a new configuration and copy the proxy url. Later this url should be added to the `.env` file in order to make the test run.

### 3. Generate certificates
Create new CA to be used by the proxy
```bash
$ cd certs
$ ./generate-certs.sh
```

### 4. Create .env file and populate the values
```bash
$ cp .env.example .env
```
Inspect the `.env` file and replace placeholders with proper values. Please note that `SECRET_KEY` should match the secret key used in the management tool.

### 5. Run and test
Start the proxy with `make run` and run a test request against `google.com` with `make test`, in the log you should be able to observe that the headers were updated according to your configurations.
E.g.:
```
2025/02/02 16:45:55 incoming proxy request:
CONNECT google.com:443 HTTP/1.1
Host: google.com:443
Proxy-Authorization: Basic ...
Proxy-Connection: Keep-Alive
User-Agent: curl/8.1.2


2025/02/02 16:45:55 incoming request:
GET / HTTP/1.1
Host: google.com
Accept: */*
User-Agent: curl/8.1.2


2025/02/02 16:45:55 updated request:
GET / HTTP/1.1
Host: google.com
Accept: */*
Authorization: my secret key
User-Agent: curl/8.1.2


2025/02/02 16:45:55 forwarding request to target...
2025/02/02 16:45:55 target response:
HTTP/1.1 200 OK
Transfer-Encoding: chunked
...
```

## Future work
This is the first iteration and hence there are ample opportunities for improvement:
- [ ] Tests
- [ ] An option to update a header only for a specific host
- [ ] Emit events to keep track of altered requests (keeping logs and complience) - these should be emitted into a kafka topic and processed async
- [ ] Use these events and create a user dashboard
