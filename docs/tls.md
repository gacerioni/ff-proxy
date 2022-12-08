# Enabling TLS
There are two ways to configure the Relay Proxy to accept HTTPS requests.

### Native TLS
You can configure the Relay Proxy to start with HTTPS enabled. This can be configured using the TLS config options. See [configuration](./configuration.md) for details. 

This does not provide every fine-grained configuration option available to secure servers. If you require more control the best option is to use a program made for this purpose, and follow the "External TLS" option below.

### External TLS
The recommended way to connect to the Relay Proxy using TLS is to place a reverse proxy such as nginx in front of the Relay Proxy. Then all connected sdks should make requests to the reverse proxy url instead of hitting the Relay Proxy directly.

![TLS Setup](images/TLS.png?raw=true)

A sample docker compose for this architecture is included in our [examples folder](../examples/tls_reverse_proxy/README.md).