# gRPC example

1) Run hello world example from grpc-go

```
git clone --depth 1 https://github.com/grpc/grpc-go/ ./tmp/grpc-go
go run ./tmp/grpc-go/examples/helloworld/greeter_server/main.go
```

2) Shoot!

```
go run ./examples/grpc -d 5s -uri 0.0.0.0:50051 -procs 6 -concurrency 12
```


