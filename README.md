# tcp-health-proxy

The TCP health proxy is a useful tool when you have a load balancer only capable of doing TCP port checks, but you want to do actual health checks against an HTTP endpoint.

This is the case when using an Amazon Network Load Balancer or GCP TCP load balancer. The service is quite simple, it performs health checks against a provided HTTP endpoint, and when the health check is successful it runs a simple echo service, and when the health check fails it turns off the echo service.

# Help
```Usage of ./bin/tcp-health-proxy:
      --bind-address string   IP address to bind to (default "0.0.0.0")
      --bind-port string      Port to listen on (default "1580")
      --check-match string    golang regex to match (https://golang.org/pkg/regexp/syntax/, escape \'s
                                use '.*' to match anything and rely only on status code
                                default pattern is case insentive, looking for ok followed by a word boundry (default "(?i)^ok\\b")
      --check-uri string      URI to check (default "http://localhost:8080/healthz")
      --log-level string      logging level (trace, debug, info, warning, fatal (default "info")
      --syslog-addr string    Syslog address and port, ex: 127.0.0.1:514 (blank for local syslog socket)
      --syslog-enable         Enable syslog messages
      --syslog-proto string   protocol to use for syslog (blank for local syslog socket)
```

In practice this is very helpful with services such as ETCD, and the Kubernetes API!
