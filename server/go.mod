module httpx-server

go 1.20

replace github.com/lucas-clemente/quic-go/internal/testdata => ./github.com/lucas-clemente/quic-go/internal/testdata

replace github.com/lucas-clemente/quic-go/internal/utils => ./github.com/lucas-clemente/quic-go/internal/utils

require (
	github.com/gorilla/mux v1.8.0
	github.com/quic-go/quic-go v0.37.0
	golang.org/x/net v0.12.0
	k8s.io/klog v1.0.0
)

require (
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/onsi/ginkgo/v2 v2.9.5 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-20 v0.3.0 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090 // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
)
