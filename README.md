# depupdates
Go Dep version checker helps to determine if some dependencies could be updated.

Works with both `ssh` and `https` repositories

# Usage

```
go get -u github.com/urakozz/depupdates
depupdates
```

Example output:

```
Current          New             Name
v1.0.22          2.0.6           github.com/cheggaaa/pb
v0.8.0           0.9.0-pre1              github.com/prometheus/client_golang
v0.2.1           0.3.0           github.com/docker/go-connections
v1.7.1           1.8.0           go.uber.org/zap
v0.20.0          0.21.0          cloud.google.com/go
v1.10.0          1.11.3          google.golang.org/grpc
v0.2.0           0.7.0           github.com/go-kit/kit
```
